package publisher

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// integration test! expects actual credentials
func TestSNSPublisher_Publish(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping due to running in short mode")
	}

	ctx := context.Background()
	LOCAL_URL := "127.0.0.1:8125"
	publisher, err := New(ctx, LOCAL_URL)
	assert.NoError(t, err)

	err = publisher.Publish(ctx, gin.H{
		"num_persisted": 101,
	})
	assert.NoError(t, err)
}
