package publisher

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	jsoniter "github.com/json-iterator/go"
	"os"
)

// TODO(zeke): env vars are getting spread out! need to clean up somehow.
const (
	TopicArnEnv = "SNS_TOPIC_ARN"
)

var (
	ErrTopicArnMissing = errors.New(TopicArnEnv + " is missing")
)

type SNSPublisher struct {
	snsClient *sns.Client
	topicArn  string
}

func (p SNSPublisher) Publish(ctx context.Context, message any) error {
	encoded, err := jsoniter.MarshalToString(message)
	if err != nil {
		return err
	}

	// don't need to set JSON anywhere, it should be fine since we're encoding.
	// the docs elaborate on when to set `MessageStructure` to `json`:
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/sns#PublishInput
	_, err = p.snsClient.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.topicArn),
		Message:  aws.String(encoded),
	})
	return err
}

// New constructs an SNSPublisher
//
// aws sdk go v2 recommends using shared credentials or config files ahead of
// using env vars. See: https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/
func New(ctx context.Context) (SNSPublisher, error) {
	cfg, err := config.LoadDefaultConfig(ctx) // uses `.aws/credentials` and `.aws/config`
	if err != nil {
		return SNSPublisher{}, err
	}
	topicArn, ok := os.LookupEnv(TopicArnEnv)
	if !ok {
		return SNSPublisher{}, ErrTopicArnMissing
	}
	return SNSPublisher{
		snsClient: sns.NewFromConfig(cfg),
		topicArn:  topicArn,
	}, nil
}
