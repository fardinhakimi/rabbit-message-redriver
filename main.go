package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/exp/slices"
)

type MessageFeedItem struct {
	message    amqp.Delivery
	exchange   string
	routingKey string
}

type MessageFeed = chan MessageFeedItem

func getXDeathHeaders(msg amqp.Delivery) amqp.Table {
	return msg.Headers["x-death"].([]interface{})[0].(amqp.Table)
}

func getOrignalMessageRoutingKey(xDeathHeaders amqp.Table) string {
	var routingKey string
	for _, value := range xDeathHeaders["routing-keys"].([]interface{}) {
		switch value.(type) {
		case string:
			if routingKey == "" {
				routingKey = fmt.Sprintf("%v", value)
				break
			}
		}
	}
	return routingKey
}

func drainDlQ(rabbitChannl *amqp.Channel, routingKeys []string) MessageFeed {
	messageFeed := make(MessageFeed, 1)
	msgs, err := rabbitChannl.Consume(
		os.Getenv("DLQ_NAME"), // queue
		"",                    // consumer
		false,                 // auto-ack
		false,                 // exclusive
		false,                 // no-local
		false,                 // no-wait
		nil,                   // args
	)

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for msg := range msgs {

			xDeathHeaders := getXDeathHeaders(msg)

			routingKey := getOrignalMessageRoutingKey(xDeathHeaders)

			if slices.Contains(routingKeys, routingKey) {
				fmt.Println("Reading message from ", msg.RoutingKey)
				messageFeed <- MessageFeedItem{
					message:    msg,
					exchange:   xDeathHeaders["exchange"].(string),
					routingKey: routingKey,
				}
			}

		}
	}()
	return messageFeed
}

func redriveMessages(messageFeed MessageFeed, rabbitChannl *amqp.Channel) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for messageFeedItem := range messageFeed {

		fmt.Printf("Publishing message to %s -> %s \n", messageFeedItem.exchange, messageFeedItem.routingKey)
		publishError := rabbitChannl.PublishWithContext(ctx,
			messageFeedItem.exchange,   // exchange
			messageFeedItem.routingKey, // routing key
			false,                      // mandatory
			false,                      // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        messageFeedItem.message.Body,
			})
		if publishError != nil {
			fmt.Println("Nack: PUBLISH ERROR")
			messageFeedItem.message.Nack(false, true)
		} else {
			fmt.Println("Message Published Successfully")
			messageFeedItem.message.Ack(false)
		}
	}
}

func main() {

	fmt.Println("Program started.")

	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	dlqName := os.Getenv("DLQ_NAME")
	routingKeys := strings.Split(os.Getenv("ROUTING_KEYS"), ",")

	if len(routingKeys) == 0 || rabbitmqURL == "" || dlqName == "" {
		log.Fatal("Program is not configured correctly")
	}

	rabbitConn, err := amqp.Dial(rabbitmqURL)

	if err != nil {
		log.Fatal(err)
	}

	defer rabbitConn.Close()

	ch, err := rabbitConn.Channel()

	if err != nil {
		log.Fatal(err)
	}

	defer ch.Close()

	// This is just for being able to run it locally.
	// The program is mainly designed to work against a cloud based rabbit-mq server which already
	// has the exchanges and queues configured and just act as a message redriver.
	_, err = ch.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	messageFeed := drainDlQ(ch, routingKeys)
	go redriveMessages(messageFeed, ch)
	sig := <-sigs
	fmt.Printf("Received signal: %s", sig.String())
	fmt.Println("Program ended.")
}
