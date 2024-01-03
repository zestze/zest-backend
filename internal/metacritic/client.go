package metacritic

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/samber/lo"
)

func FetchPosts(ctx context.Context, opts Options) ([]Post, error) {
	// make network request to metacritic!
	uri := "https://www.metacritic.com/browse/" + opts.Medium.ToPath() + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		slog.Error("error generating request", "error", err)
		return nil, err
	}

	q := req.URL.Query()
	q.Add("releaseYearMin", strconv.Itoa(opts.MinYear))
	q.Add("releaseYearMax", strconv.Itoa(opts.MaxYear))
	q.Add("page", strconv.Itoa(opts.Page))
	req.URL.RawQuery = q.Encode()

	// TODO(zeke): don't use DefaultClient!
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("error making request", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	cards, err := Extract(ctx, resp.Body)
	if err != nil {
		slog.Error("error extracting product card info from html body", "error", err)
		return nil, err
	}

	now := time.Now().UTC()
	return lo.Map(cards, func(card ProductCard, _ int) Post {
		return Post{
			ProductCard: card,
			Medium:      opts.Medium,
			RequestedAt: now,
		}
	}), nil
}

func Extract(ctx context.Context, body io.ReadCloser) ([]ProductCard, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}

	// can do _info for a closer example (to the content I'm looking for!)
	cards := make([]ProductCard, 0)
	doc.Find(".c-finderProductCard_container").Each(func(_ int, s *goquery.Selection) {
		if c, ok := newProductCard(s); ok {
			cards = append(cards, c)
		}
	})

	return cards, nil
}

// figured this out by pulling down index.html and looking and inspecting the cards.
// can see what the `Properties` are and figure out the rest from there!`
func newProductCard(s *goquery.Selection) (ProductCard, bool) {
	href, ok := s.Attr("href")
	if !ok {
		slog.Warn("did not find href attribute in card")
		return ProductCard{}, false
	}

	logger := slog.With("href", href)

	title, ok := s.Find(".c-finderProductCard_title").First().Attr("data-title")
	if !ok {
		logger.Warn("did not find title in card")
		return ProductCard{}, false
	}

	rawScore := s.Find(".c-finderProductCard_metascoreValue").First().Text()
	score, err := strconv.Atoi(rawScore)
	if err != nil {
		logger.Warn("could not convert metascore value in card")
		return ProductCard{}, false
	}

	desc := s.Find(".c-finderProductCard_description").First().Text()

	// this is super brittle. At the time of making, the structure is roughly:
	// <div class="c-finderProductCard_meta">
	// 		<span class="u-text-uppercase"> Mar 6, 2023 </span>
	//      <span> &nbsp;*&nbsp; </span>
	//		<span>
	//			<span class="u-text-capitalize"...
	//		</span>
	// so opting to look for first child. In the future can do something else.
	// </div>
	rawDate := s.Find(".c-finderProductCard_meta").First().Children().First().Text()
	rawDate = strings.TrimSpace(rawDate)
	date, err := time.Parse("Jan 2, 2006", rawDate)
	if err != nil {
		// try parsing as just a year!
		// saw errors like:
		// !BADKEY="parsing time \"2022\" as \"Jan 2, 2006\": cannot parse \"2022\" as \"Jan\""
		year, err2 := strconv.Atoi(rawDate)
		if err2 != nil {
			logger.Warn("could not parse date in card",
				"error1", err, "error2", err2)
			return ProductCard{}, false
		}
		date = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	return ProductCard{
		Title:       title,
		Href:        href,
		Score:       score,
		Description: desc,
		ReleaseDate: date,
	}, true
}
