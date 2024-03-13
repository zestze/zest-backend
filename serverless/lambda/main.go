package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	jsoniter "github.com/json-iterator/go"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"log/slog"
	"os"
)

func NewTwilioClient() (*twilio.RestClient, error) {
	id, ok := os.LookupEnv(IdEnv)
	if !ok {
		return nil, ErrAccountSidMissing
	}

	apiKey, ok := os.LookupEnv(KeyEnv)
	if !ok {
		return nil, ErrApiKeyMissing
	}

	apiSecret, ok := os.LookupEnv(SecretEnv)
	if !ok {
		return nil, ErrApiSecretMissing
	}

	return twilio.NewRestClientWithParams(twilio.ClientParams{
		AccountSid: id,
		Username:   apiKey,
		Password:   apiSecret,
	}), nil
}

func NewMessage(body string) (*twilioApi.CreateMessageParams, error) {
	to, ok := os.LookupEnv(DestEnv)
	if !ok {
		return nil, ErrDestMissing
	}
	from, ok := os.LookupEnv(OriginEnv)
	if !ok {
		return nil, ErrOriginMissing
	}
	return &twilioApi.CreateMessageParams{
		From: &from,
		To:   &to,
		Body: &body,
	}, nil
}

type SpotifyUpdate struct {
	NumPersisted int `json:"num_persisted"`
}

type Texter interface {
	CreateMessage(params *twilioApi.CreateMessageParams) (*twilioApi.ApiV2010Message, error)
}

func HandleMessage(ctx context.Context, texter Texter, message SpotifyUpdate) error {
	msg, err := NewMessage(fmt.Sprintf("%v songs persisted to zest backend!", message.NumPersisted))
	if err != nil {
		return err
	}
	_, err = texter.CreateMessage(msg)
	return err
}

// HandleEvent is the entry point for the lambda function
// for elaboration on SQS, see:
// https://github.com/aws/aws-lambda-go/blob/main/events/README_SQS.md
// for elaboration on SNS, see:
// https://github.com/aws/aws-lambda-go/blob/main/events/README_SNS.md
func HandleEvent(ctx context.Context, snsEvent events.SNSEvent) error {
	logger := slog.Default()
	err := SetupEnv()
	if err != nil {
		logger.Error("error setting up env", "error", err)
		return err
	}
	client, err := NewTwilioClient()
	if err != nil {
		logger.Error("error setting up twilio client", "error", err)
		return err
	}
	for _, record := range snsEvent.Records {
		var u SpotifyUpdate
		err := jsoniter.UnmarshalFromString(record.SNS.Message, &u)
		if err != nil {
			logger.Error("error marshaling message", "error", err)
			return err
		}
		err = HandleMessage(ctx, client.Api, u)
		if err != nil {
			logger.Error("error handling message", "error", err)
			return err
		}
	}
	return nil
}

func main() {
	lambda.Start(HandleEvent)
}
