package httptest

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func MockRTWithFile(t *testing.T, filename string) RoundTripFunc {
	return RoundTripFunc(
		func(req *http.Request) (*http.Response, error) {
			t.Helper()
			bs, err := os.ReadFile(filename)
			assert.NoError(t, err)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(bs)),
			}, nil
		})
}
