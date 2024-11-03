package publisher

import (
	"context"

	"github.com/DataDog/datadog-go/v5/statsd"
)

type DDMetricsPublisher struct {
	client *statsd.Client
}

// TODO(zeke): not sure on having a strict or flexible API for `message`
func (p DDMetricsPublisher) Publish(ctx context.Context, message any) error {
	// TODO(zeke): want this to be more descriptive!
	return p.client.Event(&statsd.Event{
		Title: "refresh",
		Text:  "songs persisted",
		// TODO(zeke): set these?
		//Tags: []string{},
		//SourceTypeName: "",
		//Hostname: "",
	})
}

// TODO(zeke): need to resolve using docker compose instead, also use env var, also enable dogstatsd
// for dd agent
func New(ctx context.Context, dogStatsdURL string) (DDMetricsPublisher, error) {
	statsd, err := statsd.New(dogStatsdURL)
	if err != nil {
		return DDMetricsPublisher{}, err
	}
	return DDMetricsPublisher{
		client: statsd,
	}, nil
}
