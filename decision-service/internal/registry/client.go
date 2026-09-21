package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"example.com/policy-snapshot/decision-service/internal/domain"
)

var ErrNotModified = errors.New("policy bundle not modified")

type HTTPStatusError struct {
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("registry returned HTTP %d", e.StatusCode)
}

type ResponseTooLargeError struct {
	Limit int64
}

func (e *ResponseTooLargeError) Error() string {
	return fmt.Sprintf("registry response exceeded %d bytes", e.Limit)
}

type ManifestResult struct {
	Manifest  domain.Manifest
	ETag      string
	BytesRead int64
}

type PageResult struct {
	Page      domain.BundlePage
	BytesRead int64
}

type Client struct {
	baseURL          string
	httpClient       *http.Client
	maxResponseBytes int64
}

func NewClient(baseURL string, httpClient *http.Client, maxResponseBytes int64) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("policy registry URL must be absolute")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if maxResponseBytes <= 0 {
		return nil, errors.New("maximum response size must be positive")
	}
	return &Client{
		baseURL:          strings.TrimRight(baseURL, "/"),
		httpClient:       httpClient,
		maxResponseBytes: maxResponseBytes,
	}, nil
}

func (c *Client) FetchManifest(ctx context.Context, ifNoneMatch string) (ManifestResult, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/policy-bundles/current", nil)
	if err != nil {
		return ManifestResult{}, err
	}
	if ifNoneMatch != "" {
		request.Header.Set("If-None-Match", ifNoneMatch)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return ManifestResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotModified {
		return ManifestResult{}, ErrNotModified
	}
	if response.StatusCode != http.StatusOK {
		return ManifestResult{}, &HTTPStatusError{StatusCode: response.StatusCode}
	}
	payload, err := c.readBody(response.Body)
	if err != nil {
		return ManifestResult{}, err
	}
	var manifest domain.Manifest
	if err := strictDecode(payload, &manifest); err != nil {
		return ManifestResult{}, fmt.Errorf("decode manifest: %w", err)
	}
	return ManifestResult{Manifest: manifest, ETag: response.Header.Get("ETag"), BytesRead: int64(len(payload))}, nil
}

func (c *Client) FetchPage(ctx context.Context, revision, pageIndex int) (PageResult, error) {
	path := "/v1/policy-bundles/" + strconv.Itoa(revision) + "/pages/" + strconv.Itoa(pageIndex)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return PageResult{}, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return PageResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return PageResult{}, &HTTPStatusError{StatusCode: response.StatusCode}
	}
	payload, err := c.readBody(response.Body)
	if err != nil {
		return PageResult{}, err
	}
	var page domain.BundlePage
	if err := strictDecode(payload, &page); err != nil {
		return PageResult{}, fmt.Errorf("decode page: %w", err)
	}
	return PageResult{Page: page, BytesRead: int64(len(payload))}, nil
}

func (c *Client) readBody(reader io.Reader) ([]byte, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, c.maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > c.maxResponseBytes {
		return nil, &ResponseTooLargeError{Limit: c.maxResponseBytes}
	}
	return payload, nil
}

func strictDecode(payload []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
