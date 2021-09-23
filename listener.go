package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/streadway/amqp"
)

type Junk struct {
	altitude int
	size int
}

var junks = make(chan Junk, 200)

func failOnError(err error, msg string) {
    if err != nil {
        log.Fatalf("%s: %s", msg, err)
    }
}

func junk_creator(junks) {
	for {
		
	}
}

func main() {
	// Pause while rabbitmq inits
	start_time, err := strconv.ParseInt(os.Getenv("STARTUP_TIME"), 10, 64)
	failOnError(err, "Failed to get start time")
	log.Printf("Waiting %d seconds for rabbitmq", start_time)
	time.Sleep(time.Duration(start_time) * time.Second)

    conn, err := amqp.Dial("amqp://guest:guest@rabbitmq/")
    failOnError(err, "Failed to connect to RabbitMQ")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    defer ch.Close()

	err = ch.ExchangeDeclare(
		"micro-rockets", // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	failOnError(err, "Failed to declare an exchange")

    q, err := ch.QueueDeclare(
        "junk-listener", // name
        false,   // durable
        false,   // delete when unused
        true,   // exclusive
        false,   // no-wait
        nil,     // arguments
    )
    failOnError(err, "Failed to declare a queue")

	err = ch.QueueBind(
		q.Name,        			// queue name
		"rocket.*.updated",     // routing key
		"micro-rockets", 		// exchange
		false,					// no-wait
		nil,					// arguments
	)
	failOnError(err, "Failed to bind a queue")

    msgs, err := ch.Consume(
        q.Name, // queue
        "",     // consumer
        true,   // auto-ack
        false,  // exclusive
        false,  // no-local
        false,  // no-wait
        nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    forever := make(chan bool)

    go func() {
        for d := range msgs {
            log.Printf("Received a message: %s", d.Body)
        }
    }()

    log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
    <-forever
}