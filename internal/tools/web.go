package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ashishk1331/seline-agent/internal/constants"
	"github.com/ashishk1331/seline-agent/internal/logging"
)

// webClient is the shared HTTP client for TinyFish requests.
var webClient = &http.Client{Timeout: 60 * time.Second}

// searchResult is one TinyFish search hit.
type searchResult struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet"`
	SiteName string `json:"site_name"`
}

type searchResponse struct {
	Results []searchResult `json:"results"`
}

type fetchResult struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Text        string `json:"text"`
}

type fetchResponse struct {
	Results []fetchResult `json:"results"`
}

// registerWeb registers web_search and web_fetch.
func (r *Registry) registerWeb() {
	r.Register(Tool{
		Name:          "web_search",
		Description:   "Search the web. Use this to find information or discover URLs.",
		Params:        []Param{{Name: "query", Type: "string", Description: "The search query."}},
		MaxChars:      -1,
		StatusMessage: `Searching for "$query"`,
		Handler:       r.webSearch,
	})

	r.Register(Tool{
		Name:          "web_fetch",
		Description:   "Fetch the full content of a URL as markdown. Use this when you already have a URL.",
		Params:        []Param{{Name: "url", Type: "string", Description: "The URL to fetch."}},
		MaxChars:      8000,
		StatusMessage: "Fetching $url",
		Handler:       r.webFetch,
	})
}

func (r *Registry) webSearch(ctx context.Context, args map[string]any) (string, error) {
	query := argString(args, "query")
	q := url.Values{}
	q.Set("query", query)
	endpoint := fmt.Sprintf("%s?%s", r.cfg.TinyfishSearchURL, q.Encode())

	body, status, err := r.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil || status != http.StatusOK {
		logging.Log.Error("web_search failed", "status", status, "err", err, "body", string(body))
		return fmt.Sprintf("Failed to search for %s", query), nil
	}

	var data searchResponse
	if err := json.Unmarshal(body, &data); err != nil {
		logging.Log.Error("web_search decode failed", "err", err)
		return fmt.Sprintf("Failed to search for %s", query), nil
	}

	siteNames := make([]string, 0, len(data.Results))
	for _, res := range data.Results {
		siteNames = append(siteNames, strings.TrimPrefix(res.SiteName, "www."))
	}
	logging.Log.Info("web_search", "query", query, "sites", strings.Join(topThree(siteNames), ", "))

	lines := make([]string, 0, len(data.Results))
	for _, res := range data.Results {
		lines = append(lines, fmt.Sprintf("[%s](%s) - %s", res.Title, res.URL, res.Snippet))
	}
	return strings.Join(lines, "\n"), nil
}

func (r *Registry) webFetch(ctx context.Context, args map[string]any) (string, error) {
	target := argString(args, "url")
	payload, _ := json.Marshal(map[string]any{
		"urls":   []string{target},
		"format": "markdown",
	})

	body, status, err := r.doRequest(ctx, http.MethodPost, r.cfg.TinyfishFetchURL, payload)
	if err != nil || status != http.StatusOK {
		logging.Log.Error("web_fetch failed", "status", status, "err", err, "body", string(body))
		return fmt.Sprintf("Failed to fetch %s", target), nil
	}

	var data fetchResponse
	if err := json.Unmarshal(body, &data); err != nil || len(data.Results) == 0 {
		logging.Log.Error("web_fetch decode failed", "err", err)
		return fmt.Sprintf("Failed to fetch %s", target), nil
	}

	res := data.Results[0]
	desc := res.Description
	if desc == "" {
		desc = res.Title
	}
	logging.Log.Info("web_fetch", "url", target, "desc", desc)
	return res.Text, nil
}

// doRequest performs an HTTP request with the TinyFish headers and returns the
// body, status code and any transport error.
func (r *Registry) doRequest(ctx context.Context, method, endpoint string, body []byte) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range constants.TinyFishHeaders(r.cfg) {
		req.Header.Set(k, v)
	}

	resp, err := webClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, err
}
