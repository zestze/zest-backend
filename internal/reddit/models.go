package reddit

type Secrets struct {
	ClientId     string
	ClientSecret string
	Username     string
	Password     string
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type Child struct {
	Subreddit           string
	Permalink           string
	NumComments         int     `json:"num_comments"`
	UpvoteRatio         float64 `json:"upvote_ratio"`
	Ups                 int
	Score               int
	TotalAwardsReceived int    `json:"total_awards_received"`
	SuggestedSort       string `json:"suggested_sort"`
}

type ApiResponse struct {
	Data struct {
		Children []struct {
			Data Child
		}
		After string
	}
}
