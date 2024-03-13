package publisher

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"testing"
)

// integration test! expects actual credentials
func TestSNSPublisher_Publish(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping due to running in short mode")
	}

	t.Setenv(TopicArnEnv, "arn:aws:sns:us-east-1:862873347672:spotify-update")
	ctx := context.Background()
	publisher, err := New(ctx)
	assert.NoError(t, err)

	err = publisher.Publish(ctx, gin.H{
		"num_persisted": 101,
	})
	assert.NoError(t, err)
}
