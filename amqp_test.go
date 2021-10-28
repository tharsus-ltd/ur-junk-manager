package main

import (
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/streadway/amqp"
)


func TestCreateMessage(t *testing.T) {

	tracer, closer := Init("Junk Manager", true)
    defer closer.Close()

	span := tracer.StartSpan(
        "rocket.updated",
    )
    defer span.Finish()

	msg := amqp.Publishing{
        ContentType: "text/json",
        Body:        []byte("test,test,test"),
        Headers:     map[string]interface{}{},
    }
	headers := amqpHeadersCarrier(msg.Headers)

	tracer.Inject(
        span.Context(),
        opentracing.TextMap,
        headers,
    )
}