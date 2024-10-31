package rabbitmq

import (
	"encoding/json"
	"log"

	"github.com/codecoogs/gogo/constants" // Update this import path to match your project

	"github.com/rabbitmq/amqp091-go"
)

func PublishPurchaseEvent(ch *amqp091.Channel, q amqp091.Queue, purchase constants.PurchaseEvent) {
	body, err := json.Marshal(purchase)
	if err != nil {
		log.Fatalf("Failed to marshal purchase event: %s", err)
	}

	err = ch.Publish(
		"",     // Exchange
		q.Name, // Routing key (queue name)
		false,  // Mandatory
		false,  // Immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		log.Fatalf("Failed to publish purchase event: %s", err)
	}
	log.Printf("Published purchase event for user: %s", purchase.UserEmail)
}
