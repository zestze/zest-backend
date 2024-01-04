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

// TODO(zeke): store id!
type Post struct {
	Subreddit           string  `json:"subreddit"`
	Permalink           string  `json:"permalink"`
	NumComments         int     `json:"num_comments"`
	UpvoteRatio         float64 `json:"upvote_ratio"`
	Ups                 int     `json:"ups"`
	Score               int     `json:"score"`
	TotalAwardsReceived int     `json:"total_awards_received"`
	SuggestedSort       string  `json:"suggested_sort"`

	// recently added
	Title      string  `json:"title,omitempty"`
	Name       string  `json:"name,omitempty"`        // appears to be "thing type" + "id"
	CreatedUTC float64 `json:"created_utc,omitempty"` // appears to be an epoch float
}

type ApiResponse struct {
	Data struct {
		Children []struct {
			Data Post
		}
		After string
	}
}
