package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
)

const (
	libraryVersion = "1.60.0"
	defaultBaseURL = "https://api.digitalocean.com/"
	userAgent      = "godo/" + libraryVersion
	mediaType      = "application/json"

	headerRateLimit     = "RateLimit-Limit"
	headerRateRemaining = "RateLimit-Remaining"
	headerRateReset     = "RateLimit-Reset"
)

type Client struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	// Base URL for API requests.
	BaseURL *url.URL

	// User agent for client
	UserAgent string

	// Rate contains the current rate limit for the client as determined by the most recent
	// API call. It is not thread-safe. Please consider using GetRate() instead.
	Rate    Rate
	ratemtx sync.Mutex

	// Services used for communicating with the API
	Tags TagsService

	// Optional extra HTTP headers to set on every request to the API.
	headers map[string]string
}

type ListOptions struct {
	// For paginated result sets, page of results to retrieve.
	Page int `url:"page,omitempty"`

	// For paginated result sets, the number of results to include per page.
	PerPage int `url:"per_page,omitempty"`
}

// Rate contains the rate limit for the current client.
type Rate struct {
	// The number of request per hour the client is currently limited to.
	Limit int `json:"limit"`

	// The number of remaining requests the client can make this hour.
	Remaining int `json:"remaining"`

	// The time at which the current rate limit will reset.
	Reset Timestamp `json:"reset"`
}

// Response is a DigitalOcean response. This wraps the standard http.Response returned from DigitalOcean.
type Response struct {
	*http.Response

	Rate
}

/* NEW CLIENT */

// NewClient returns a new DigitalOcean API client, using the given
// http.Client to perform all requests.
//
// Users who wish to pass their own http.Client should use this method. If
// you're in need of further customization, the godo.New method allows more
// options, such as setting a custom URL or a custom user agent string.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	baseURL, _ := url.Parse(defaultBaseURL)

	c := &Client{client: httpClient, BaseURL: baseURL, UserAgent: userAgent}
	c.Tags = &TagsServiceOp{client: c}

	return c
}

// New returns a new DigitalOcean API client instance.
func New(httpClient *http.Client) (*Client, error) {
	c := NewClient(httpClient)

	return c, nil
}

/* NEW REQUEST */

func (c *Client) NewRequest(ctx context.Context, method, urlStr string, body interface{}) (*http.Request, error) {
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		req, err = http.NewRequest(method, u.String(), nil)
		if err != nil {
			return nil, err
		}
	case http.MethodPost:
		buf := new(bytes.Buffer)
		if body != nil {
			err = json.NewEncoder(buf).Encode(body)
			if err != nil {
				return nil, err
			}
		}

		req, err = http.NewRequest(method, u.String(), buf)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", mediaType)
	default:
		//
	}
	// add headers
	req.Header.Set("Accept", mediaType)
	req.Header.Set("User-Agent", c.UserAgent)

	return req, nil
}
