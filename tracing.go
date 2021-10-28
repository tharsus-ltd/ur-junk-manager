package main

import (
	"io"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	config "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/log"
)

// amqpHeadersCarrier satisfies both TextMapWriter and TextMapReader.
//
// Example usage for server side:
//
//     carrier := amqpHeadersCarrier(amqp.Table)
//     clientContext, err := tracer.Extract(
//         opentracing.TextMap,
//         carrier)
//
// Example usage for client side:
//
//     carrier := amqpHeadersCarrier(amqp.Table)
//     err := tracer.Inject(
//         span.Context(),
//         opentracing.TextMap,
//         carrier)
//
type amqpHeadersCarrier map[string]interface{}


func (c amqpHeadersCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, val := range c {
		v, ok := val.(string)
		if !ok {
			continue
		}
		if err := handler(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (c amqpHeadersCarrier) Set(key, val string) {
	c[key] = val
}

func Init(service string, testing bool) (opentracing.Tracer, io.Closer) {

	if (testing) {
		tracer, closer := jaeger.NewTracer(
			"TestService",
			jaeger.NewConstSampler(true),
			jaeger.NewInMemoryReporter(),
			jaeger.TracerOptions.Metrics(jaeger.NewNullMetrics()),
			jaeger.TracerOptions.Logger(log.NullLogger),
		)
		return tracer, closer
	}

	cfg, err := config.FromEnv()
	if err != nil {
		panic("Could not parse Jaeger env vars: " + err.Error())
	}

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		panic("Could not initialize jaeger tracer: " + err.Error())
	}
	return tracer, closer
}