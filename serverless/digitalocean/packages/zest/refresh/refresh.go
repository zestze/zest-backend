package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"slices"
	"time"

	jsoniter "github.com/json-iterator/go"
)

/*
type Response struct {
}
*/

type Event struct {
	// Resource should be set to `metacritic` or `reddit` or `spotify`
	Resource string `json:"resource"`
}

func (e Event) Valid() bool {
	options := [3]string{"reddit", "metacritic", "spotify"}
	return slices.Contains(options[:], e.Resource)
}

func Main(ctx context.Context, event Event) {
	logger := log.Default()

	if !slices.Contains([]string{"reddit", "metacritic", "spotify"}, event.Resource) {
		logger.Fatal("invalid event type: ", event.Resource)
		return
	}

	// login first!
	jar, err := cookiejar.New(nil)
	if err != nil {
		logger.Fatal("error making cookie jar: ", err)
		return
	}

	var bs bytes.Buffer
	if err := jsoniter.NewEncoder(&bs).Encode(struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: os.Getenv("ZEST_USERNAME"),
		Password: os.Getenv("ZEST_PASSWORD"),
	}); err != nil {
		logger.Fatal("error encoding credentials: ", err)
		return
	}

	logger.Print("logging into server")
	uri := "https://api.zekereyna.dev/login"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, &bs)
	if err != nil {
		logger.Fatal("error making login request: ", err)
		return
	}

	client := http.Client{
		Timeout: 60 * time.Second,
		Jar:     jar,
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Fatal("error doing login request: ", err)
		return
	}
	resp.Body.Close() // nothing to handle but still should close
	if resp.StatusCode != http.StatusOK {
		logger.Fatal("error code from login request, status code:", resp.StatusCode)
		return
	}

	uri = "https://api.zekereyna.dev/v1/" + event.Resource + "/refresh"
	logger.Print("refreshing server, uri: ", uri)
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, uri, nil)
	if err != nil {
		logger.Fatal("error making request: ", err)
		return
	}

	resp, err = client.Do(req)
	if err != nil {
		logger.Fatal("error doing request, err: ", err)
		return
	}
	resp.Body.Close() // nothing to handle but still should close
	if resp.StatusCode != http.StatusCreated {
		logger.Fatal("error doing request, status code: ", resp.StatusCode)
		return
	}

	logger.Print("successfully refreshed server")
}

/*
func main() {
	Main(context.Background())
}
*/
