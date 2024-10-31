package rabbitmq

import (
	"log"
	"os"

	"github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// ConnectRabbitMQ establishes a connection to RabbitMQ
func ConnectRabbitMQ() *amqp091.Connection {
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	conn, err := amqp091.Dial(rabbitmqURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	return conn
}

// SetupChannel opens a channel and declares a queue
func SetupChannel(conn *amqp091.Connection) (*amqp091.Channel, amqp091.Queue) {
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")

	q, err := ch.QueueDeclare(
		"testQueue", // Queue name
		false,       // Durable
		false,       // Delete when unused
		false,       // Exclusive
		false,       // No-wait
		nil,         // Arguments
	)
	failOnError(err, "Failed to declare a queue")
	return ch, q
}

func SetupPurchaseQueue(ch *amqp091.Channel) amqp091.Queue {
	q, err := ch.QueueDeclare(
		"purchaseQueue", // Queue name
		true,            // Durable
		false,           // Delete when unused
		false,           // Exclusive
		false,           // No-wait
		nil,             // Arguments
	)
	failOnError(err, "Failed to declare a purchase queue")
	return q
}
