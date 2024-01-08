package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

/*
type Response struct {
}
*/

type Event struct {
	// Resource should be set to `metacritic` or `reddit`
	Resource string `json:"resource"`
}

func Main(ctx context.Context, event Event) {
	uri := "https://api.zekereyna.dev/v1/" + event.Resource + "/refresh"
	fmt.Println("going to refresh server at: ", uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, nil)
	// TODO(zeke): use structured logging like slog!
	if err != nil {
		fmt.Println("error making request: ", err)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error doing request: ", err)
	} else if resp.StatusCode != http.StatusOK {
		fmt.Println("error, status code is: ", resp.StatusCode)
	}

	fmt.Println("done refreshing server")
}

/*
func main() {
	Main(context.Background())
}
*/
