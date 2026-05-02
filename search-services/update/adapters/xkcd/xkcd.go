package xkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"yadro.com/course/update/core"
)

type Client struct {
	log    *slog.Logger
	client http.Client
	url    string
}

type XKCDAPI struct {
	Num        int    `json:"num"`
	Img        string `json:"img"`
	Title      string `json:"title"`
	Alt        string `json:"alt"`
	Month      string `json:"month"`
	Link       string `json:"link"`
	Year       string `json:"year"`
	News       string `json:"news"`
	Safe_title string `json:"safe_title"`
	Transcript string `json:"transcript"`
}

func (c *XKCDAPI) ToXKCDInfo() core.XKCDInfo {
	return core.XKCDInfo{
		ID:          c.Num,
		URL:         c.Img,
		Title:       c.Title,
		Description: c.Alt,
		Transcript:  c.Transcript,
	}
}

func NewClient(url string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	return &Client{
		client: http.Client{Timeout: timeout},
		log:    log,
		url:    url,
	}, nil
}

func (c *Client) Get(ctx context.Context, id int) (core.XKCDInfo, error) {
	var pd XKCDAPI
	path := fmt.Sprintf("%s/%d/info.0.json", c.url, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		c.log.Error("Request failed", "error", err)
		return core.XKCDInfo{}, core.ErrBadArguments
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return core.XKCDInfo{}, core.ErrBadArguments
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return core.XKCDInfo{}, core.ErrBadArguments
	}
	if resp.StatusCode != http.StatusOK {
		return core.XKCDInfo{}, core.ErrBadArguments
	}
	if err = json.NewDecoder(resp.Body).Decode(&pd); err != nil {
		return core.XKCDInfo{}, core.ErrBadArguments
	}
	return pd.ToXKCDInfo(), nil
}

func (c *Client) LastID(ctx context.Context) (int, error) {
	var pd XKCDAPI
	path := fmt.Sprintf("%s/info.0.json", c.url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		c.log.Error("Request failed", "error", err)
		return 0, core.ErrBadArguments
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, core.ErrBadArguments
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, core.ErrBadArguments
	}
	if err := json.NewDecoder(resp.Body).Decode(&pd); err != nil {
		return 0, core.ErrBadArguments
	}
	return pd.Num, nil
}
