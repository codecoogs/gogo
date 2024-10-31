package rabbitmq

import (
	"fmt"
	"log"

	"github.com/rabbitmq/amqp091-go"
)

// PublishMessage sends a message to the specified queue
func PublishMessage(ch *amqp091.Channel, q amqp091.Queue, body string) {
	err := ch.Publish(
		"",     // Exchange
		q.Name, // Routing key (queue name)
		false,  // Mandatory
		false,  // Immediate
		amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	if err != nil {
		log.Fatalf("Failed to publish a message: %s", err)
	}
	fmt.Printf("Sent: %s\n", body)
}
