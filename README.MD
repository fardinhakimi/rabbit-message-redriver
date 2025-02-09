# Rabbit Message Redriver

This program is designed to drain a DLQ and put back the messages to the original queue.

This program is designed to be used when the problem that caused the messages to be **dead-lettered** is fixed on the **service**, otherwise you might endup in an infinite loop!

Furthermore, this program is designed in a way that expects the queues and routing keys to already exist on the RabbitMQ server.

----
## Up and Running Guide

#### Setup:
- Install [Golang](https://go.dev/doc/install) on your local machine
- Run `go mod tidy` to download dependencies
- Start local RabbitMQ server and management UI by running ```docker-compose up -d ``` from project root.
- You can access [RabbitMQ Management UI](http://0.0.0.0:15672/#/)
- Use **test** as **username** and **password** for **RABBITMQ_URL** and **RabbitMQ Management UI**.
- Create a `.env` file at the root of this project see `.env.example` on what to put in the file.

#### Running the Program
- Run by `go run main.go` or alternatively first compile by `go build main.go` and then run by `./main` in the terminal to run the program. The program listens to the DLQ for messages and puts them back to the original queue. The program runs untill you explicity end the program.
- Use `ctrl+c` to gracefully end the program.