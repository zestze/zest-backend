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

func Main(ctx context.Context) {
	fmt.Println("going to refresh server")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://164.90.252.244/v1/reddit/refresh", nil)
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
