package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"

	"github.com/codecoogs/gogo/constants"

	"github.com/rabbitmq/amqp091-go"
)

func ConsumePurchaseQueue(ch *amqp091.Channel, q amqp091.Queue) {
	msgs, err := ch.Consume(
		q.Name, // Queue name
		"",     // Consumer tag
		true,   // Auto-ack
		false,  // Exclusive
		false,  // No-local
		false,  // No-wait
		nil,    // Args
	)
	failOnError(err, "Failed to register a consumer for purchase queue")

	go func() {
		for d := range msgs {
			var purchase constants.PurchaseEvent
			if err := json.Unmarshal(d.Body, &purchase); err != nil {
				log.Printf("Error decoding message: %s", err)
				continue
			}
			sendEmail(purchase)
		}
	}()
}

// Simple email-sending function (using Gmail SMTP for illustration)
func sendEmail(purchase constants.PurchaseEvent) {
	from := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASS")
	to := []string{purchase.UserEmail}

	// SMTP server configuration
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	message := []byte(fmt.Sprintf("To: %s\r\nSubject: Purchase Confirmation\r\n\r\nDear %s, thank you for purchasing %s for $%.2f.",
		purchase.UserEmail, purchase.UserName, purchase.ProductName, purchase.PurchasePrice))

	auth := smtp.PlainAuth("", from, password, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		log.Printf("Failed to send email to %s: %s", purchase.UserEmail, err)
	} else {
		log.Printf("Sent email to %s", purchase.UserEmail)
	}
}
