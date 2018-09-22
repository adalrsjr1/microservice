package main

import (
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	reporterhttp "github.com/openzipkin/zipkin-go/reporter/http"
)

func newTracer(serviceName string, zipkinEndpoint string) (*zipkin.Tracer, error) {
	endpointURL := "http://" + zipkinEndpoint + "/api/v2/spans"

	// The reporter sends traces to zipkin server
	reporter := reporterhttp.NewReporter(endpointURL)

	// Local endpoint represent the local service information
	localEndpoint := &model.Endpoint{ServiceName: serviceName, Port: 8080}

	// Sampler tells you which traces are going to be sampled or not. In
	// this case we will record 100% (1.00) of traces
	sampler, err := zipkin.NewCountingSampler(1)
	if err != nil {
		return nil, err
	}

	t, err := zipkin.NewTracer(
		reporter,
		zipkin.WithSampler(sampler),
		zipkin.WithLocalEndpoint(localEndpoint),
	)

	if err != nil {
		return nil, err
	}

	return t, err
}
