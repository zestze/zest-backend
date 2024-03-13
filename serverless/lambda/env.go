package main

import (
	"errors"
	"github.com/joho/godotenv"
	"os"
)

const (
	IdEnv     = "TWILIO_ACCOUNT_SID"
	KeyEnv    = "TWILIO_API_KEY"
	SecretEnv = "TWILIO_API_SECRET"
	OriginEnv = "TWILIO_ORIGIN_NUMBER"
	DestEnv   = "TWILIO_DEST_NUMBER"
)

var (
	ErrAccountSidMissing = EnvMissing(IdEnv)
	ErrApiKeyMissing     = EnvMissing(KeyEnv)
	ErrApiSecretMissing  = EnvMissing(SecretEnv)
	ErrOriginMissing     = EnvMissing(OriginEnv)
	ErrDestMissing       = EnvMissing(DestEnv)
)

func EnvMissing(envKey string) error {
	return errors.New(envKey + " is missing")
}

// SetupEnv loads .env file if it exists
func SetupEnv() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	} // swallow ErrNotExist
	return nil
}
