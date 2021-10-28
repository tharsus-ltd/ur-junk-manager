package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/streadway/amqp"
)

type Rocket struct {
	Num_engines  int `json:"num_engines"`
	Height       int `json:"height"`
	Id           string `json:"id"`
	Fuel         float32 `json:"fuel"`
	Altitude     float32 `json:"altitude"`
	Velocity     float32 `json:"velocity"`
	Crashed      bool `json:"crashed"`
	Launched     bool `json:"launched"`
	Max_altitude float32 `json:"max_altitude"`
	Status       string `json:"status"`
}

type Event struct {
	Rocket   Rocket `json:"rocket"`
	Username string `json:"username"`
}


func failOnError(err error, msg string) {
    if err != nil {
        log.Fatalf("%s: %s", msg, err)
    }
}

func handleMessage(tracer opentracing.Tracer, d amqp.Delivery, r1 *rand.Rand, ch *amqp.Channel) {
    var evt Event
            
    carrier := amqpHeadersCarrier(d.Headers)
    wireCtx, wireErr := tracer.Extract(
        opentracing.TextMap,
        carrier,
    )
    if wireErr != nil {
        failOnError(wireErr, "Failed to get header context from message")
    }

    span := tracer.StartSpan(
        "rocket.updated",
        ext.RPCServerOption(wireCtx),
    )
    defer span.Finish()

    err := json.Unmarshal(d.Body, &evt)
    failOnError(err, "Failed to decode rocket.updated event")

    junk_prob, err := strconv.ParseInt(os.Getenv("JUNK_PROBABILITY"), 10, 64)
	failOnError(err, "Failed to get JUNK_PROBABILITY")

    junk_alt_min, err := strconv.ParseFloat(os.Getenv("JUNK_ALT_MIN"), 32)
	failOnError(err, "Failed to get JUNK_ALT_MIN")

    junk_alt_max, err := strconv.ParseFloat(os.Getenv("JUNK_ALT_MAX"), 32)
	failOnError(err, "Failed to get JUNK_ALT_MIN")

    // if the rocket is above a certain altitude, there is a
    // chance it will be hit by junk:
    // https://www.sciencedirect.com/science/article/pii/S0094576514002872
    if !evt.Rocket.Crashed && evt.Rocket.Altitude >= float32(junk_alt_min) && evt.Rocket.Altitude <= float32(junk_alt_max) {
        // var modifier int64 = int64((evt.Rocket.Height / 2000) * 100)
        if int64(r1.Intn(100)) > junk_prob {

            // junk has hit the spacecraft!
            log.Printf("Rocket: %s has hit some space junk!", evt.Rocket.Id)
            span.LogKV("event", "rocket will crash")
            
            evt.Rocket.Status = "Hit by space junk! 🛰🔥"
            sendMessage(evt, fmt.Sprintf("rocket.%s.crashed", evt.Rocket.Id), ch, tracer, span)
        }
    }
}

func sendMessage(body Event, topic string, ch *amqp.Channel, tracer opentracing.Tracer, span opentracing.Span) {

    evtJson, err := json.Marshal(body)
    failOnError(err, "Error marhsalling json data")

    msg := amqp.Publishing{
        ContentType: "text/json",
        Body:        evtJson,
        Headers:     map[string]interface{}{},
    }
    headers := amqpHeadersCarrier(msg.Headers)

    tracer.Inject(
        span.Context(),
        opentracing.TextMap,
        headers,
    )

    err = ch.Publish(
        "micro-rockets",        // exchange
        topic,                  // routing key
        false,                  // mandatory
        false,                  // immediate
        msg)
    failOnError(err, "Failed to publish a message")

}

func main() {
    
    tracer, closer := Init("Junk Manager", false)
    defer closer.Close()

	// Pause while rabbitmq inits
	start_time, err := strconv.ParseInt(os.Getenv("STARTUP_TIME"), 10, 64)
	failOnError(err, "Failed to get STARTUP_TIME")

	log.Printf("Waiting %d seconds for rabbitmq", start_time)
	time.Sleep(time.Duration(start_time) * time.Second)

    s1 := rand.NewSource(time.Now().UnixNano())
    r1 := rand.New(s1)

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
            handleMessage(tracer, d, r1, ch)
        }
    }()

    log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
    <-forever
}