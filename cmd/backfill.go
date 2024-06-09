package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"slices"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type RangeParams struct {
	Start string `short:"s" help:"start of backfill if necessesary"`
	End   string `short:"e" help:"end of backfill if necessary"`
}

type BackfillCmd struct {
	Username string `short:"u" env:"ZEST_USERNAME" help:"username to sign into backend"`
	Password string `short:"p" env:"ZEST_PASSWORD" help:"password to sign into backend"`
	Resource string `short:"r" help:"resource to hit"`
	RangeParams
}

func (r *BackfillCmd) Run() error {
	ctx := context.Background()
	logger := log.Default()

	if !slices.Contains([]string{"reddit", "metacritic", "spotify"}, r.Resource) {
		logger.Fatal("invalid event type: ", r.Resource)
	}

	// login first!
	jar, err := cookiejar.New(nil)
	if err != nil {
		logger.Fatal("error making cookie jar: ", err)
	}

	var bs bytes.Buffer
	if err := jsoniter.NewEncoder(&bs).Encode(struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: r.Username,
		Password: r.Password,
	}); err != nil {
		logger.Fatal("error encoding credentials: ", err)
	}

	logger.Print("logging into server")
	uri := "https://api.zekereyna.dev/login"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, &bs)
	if err != nil {
		logger.Fatal("error making login request: ", err)
	}

	client := http.Client{
		Timeout: 60 * time.Second,
		Jar:     jar,
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Fatal("error doing login request: ", err)
	}
	resp.Body.Close() // nothing to handle but still should close
	if resp.StatusCode != http.StatusOK {
		logger.Fatal("error code from login request, status code:", resp.StatusCode)
	}

	uri = "https://api.zekereyna.dev/v1/" + r.Resource + "/backfill"
	if r.Resource == "spotify" {
		uri += fmt.Sprintf("?start=%v&end=%v", r.Start, r.End)
	}
	logger.Print("refreshing server, uri: ", uri)
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, uri, nil)
	if err != nil {
		logger.Fatal("error making request: ", err)
	}

	resp, err = client.Do(req)
	if err != nil {
		logger.Fatal("error doing request, err: ", err)
	}
	resp.Body.Close() // nothing to handle but still should close
	if resp.StatusCode >= 400 {
		logger.Fatal("error doing request, status code: ", resp.StatusCode)
	}

	logger.Print("successfully refreshed server")

	return nil
}
