package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Simple test to verify that go.mod is setup correctly and code works e2e
func TestEvent(t *testing.T) {
	assert := assert.New(t)
	e := Event{
		Resource: "spotify",
	}
	assert.True(e.Valid())

	e.Resource = "pandora"
	assert.False(e.Valid())
}
