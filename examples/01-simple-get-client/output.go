// Code generated by gen-client -client-name MessageService -output output.go api.yaml. DO NOT EDIT.
package messageservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/vitaminniy/go-lib-http/config"
)

// This is needed to have bytes imported when non-body requests are generated.
var _ = bytes.Buffer{}

// Option overrides MessageService creation.
type Option func(*MessageService)

// WithTransport overrides the default http client transport.
func WithTransport(transport http.RoundTripper) Option {
	return func(cl *MessageService) {
		cl.httpClient.Transport = transport
	}
}

// WithTimeout overrides the default http client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(cl *MessageService) {
		cl.httpClient.Timeout = timeout
	}
}

// WithSnapshot overrides the default config snapshot.
func WithSnapshot(snapshot *config.Snapshot[Config]) Option {
	return func(cl *MessageService) {
		cl.snapshot = snapshot
	}
}

// NewMessageService creates a new MessageService http client.
func NewMessageService(baseurl string, opts ...Option) (*MessageService, error) {
	parsed, err := url.Parse(baseurl)
	if err != nil {
		return nil, fmt.Errorf("could not parse base url: %w", err)
	}

	cli := &MessageService{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: time.Second * 1, // Arbitrary value to avoid hanging forever.
		},
		snapshot: config.NewSnapshot(DefaultConfig()),
	}

	for _, opt := range opts {
		opt(cli)
	}

	return cli, nil
}

type MessageService struct {
	baseURL    *url.URL
	httpClient *http.Client
	snapshot   *config.Snapshot[Config]
}

func (cl *MessageService) getConfig() Config {
	if cl.snapshot == nil {
		return DefaultConfig()
	}

	return cl.snapshot.Get()
}

type MessagesResponseBody struct {
	Messages []Message `json:"messages"`
}

type Message struct {
	Id       string `json:"id"`
	SenderId string `json:"sender_id"`
	Text     string `json:"text"`
}

// DefaultConfig returns default configuration.
func DefaultConfig() Config {
	return Config{
		GETApiV1Messages: config.QOS{},
	}
}

// Config controls service configuration.
type Config struct {
	GETApiV1Messages config.QOS
}

type GETApiV1MessagesRequest struct {
	// HeaderUserAgent is "User-Agent" header value.
	HeaderUserAgent string
	// Headers is a list of additional headers.
	Headers map[string]string

	// QueryLimit is "limit" query parameter.
	QueryLimit string
	// QuerySenderId is "sender_id" query parameter.
	QuerySenderId *string
}

type GETApiV1MessagesResponse struct {
	Headers map[string][]string

	Body200 *MessagesResponseBody
}

func (cl *MessageService) GETApiV1Messages(
	ctx context.Context,
	request *GETApiV1MessagesRequest,
) (*GETApiV1MessagesResponse, error) {
	url := cl.baseURL.JoinPath("/api/v1/messages")
	cfg := cl.getConfig().GETApiV1Messages

	ctx, cancel := cfg.Context(ctx)
	defer cancel()

	{
		query := url.Query()

		query.Add("limit", request.QueryLimit)

		if request.QuerySenderId != nil {
			query.Add("sender_id", *request.QuerySenderId)
		}

		url.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not prepare request: %w", err)
	}

	req.Header.Add("Accept", "application/json")

	req.Header.Add("User-Agent", request.HeaderUserAgent)

	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not do http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read response with status %d: %w", resp.StatusCode, err)
		}

		return nil, fmt.Errorf("got response with status %d: %q", resp.StatusCode, string(raw))
	}

	response := &GETApiV1MessagesResponse{
		Headers: resp.Header,
	}

	if resp.StatusCode == 200 {
		var body MessagesResponseBody
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return nil, fmt.Errorf("could not decode response [%d]: %w", resp.StatusCode, err)
		}

		response.Body200 = &body

		return response, nil
	}

	return nil, fmt.Errorf("unhandled response code: %d", resp.StatusCode)
}
