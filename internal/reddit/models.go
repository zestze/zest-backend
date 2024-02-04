package reddit

import "github.com/samber/lo"

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

	// for comments
	LinkTitle string `json:"link_title,omitempty"`
	Body      string `json:"body,omitempty"`
}

type ApiResponse struct {
	Data struct {
		Children []struct {
			Data Post
		}
		After string
	}
}

func (ar ApiResponse) Posts() []Post {
	return lo.Map(ar.Data.Children, func(child struct{ Data Post }, _ int) Post {
		return child.Data
	})
}
