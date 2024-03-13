package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandler(t *testing.T) {
	assert := assert.New(t)
	err := SetupEnv()
	assert.NoError(err)

	client, err := NewTwilioClient()
	assert.NoError(err)

	msg, err := NewMessage("hello world")
	assert.NoError(err)

	_, err = client.Api.CreateMessage(msg)
	assert.NoError(err)
}
