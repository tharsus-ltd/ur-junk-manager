package main

import (
	"io"
	"os"

	opentracing "github.com/opentracing/opentracing-go"
	config "github.com/uber/jaeger-client-go/config"
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

func Init(service string) (opentracing.Tracer, io.Closer) {
	
	os.Setenv("JAEGER_SERVICE_NAME", service)
	os.Setenv("JAEGER_AGENT_HOST", "jaeger")
	os.Setenv("JAEGER_AGENT_PORT", "6831")
	os.Setenv("JAEGER_SAMPLER_TYPE", "const")
	os.Setenv("JAEGER_SAMPLER_PARAM", "1")
	os.Setenv("JAEGER_REPORTER_LOG_SPANS", "true")

	defcfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}

	cfg, err := defcfg.FromEnv()
	if err != nil {
		panic("Could not parse Jaeger env vars: " + err.Error())
	}

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		panic("Could not initialize jaeger tracer: " + err.Error())
	}
	return tracer, closer
}