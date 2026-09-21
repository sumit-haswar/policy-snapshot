package upstream

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var ErrResponseTooLarge = errors.New("upstream response too large")

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

type Client struct {
	baseURL          string
	httpClient       *http.Client
	maxResponseBytes int64
}

func NewClient(baseURL string, httpClient *http.Client, maxResponseBytes int64) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("decision service URL must be absolute")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if maxResponseBytes <= 0 {
		return nil, errors.New("maximum response size must be positive")
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient, maxResponseBytes: maxResponseBytes}, nil
}

func (c *Client) Do(ctx context.Context, method, path string, header http.Header, body []byte) (Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	copyRequestHeaders(request.Header, header)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return Response{}, err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if err != nil {
		return Response{}, err
	}
	if int64(len(payload)) > c.maxResponseBytes {
		return Response{}, ErrResponseTooLarge
	}
	return Response{StatusCode: response.StatusCode, Header: response.Header.Clone(), Body: payload}, nil
}

func (c *Client) Ready(ctx context.Context) error {
	response, err := c.Do(ctx, http.MethodGet, "/health/ready", nil, nil)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("decision service readiness returned HTTP %d", response.StatusCode)
	}
	return nil
}

func copyRequestHeaders(destination, source http.Header) {
	for _, name := range []string{"Content-Type", "Accept", "X-Request-ID"} {
		for _, value := range source.Values(name) {
			destination.Add(name, value)
		}
	}
}
