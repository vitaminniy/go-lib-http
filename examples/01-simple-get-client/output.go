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
	"github.com/vitaminniy/go-lib-http/retry"
)

// This is needed to have bytes imported when non-body requests are generated.
var _ = bytes.Buffer{}

// DefaultServiceConfig is a default configuration for the MessageService.
var DefaultServiceConfig = config.ServiceConfig{
	Default: config.QOS{
		Timeout: 1 * time.Second,
		Retry: retry.Config{
			Attempts: 2,
			Backoff:  5 * time.Millisecond,
			Jitter:   10 * time.Millisecond,
		},
	},
}

// Option overrides MessageService creation.
type Option func(*MessageService)

// WithSnapshot overrides the default snapshot.
func WithSnapshot(snapshot *config.Snapshot) Option {
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
		baseURL:    parsed,
		httpClient: http.DefaultClient,
		snapshot:   config.NewSnapshot(DefaultServiceConfig),
	}

	for _, opt := range opts {
		opt(cli)
	}

	return cli, nil
}

type MessageService struct {
	baseURL    *url.URL
	httpClient *http.Client
	snapshot   *config.Snapshot
}

func (cl *MessageService) qos(name string) config.QOS {
	if cl.snapshot == nil {
		return config.QOS{}
	}

	return cl.snapshot.QOS(name)
}

type MessagesResponseBody struct {
	Messages []Message `json:"messages"`
}

type Message struct {
	Id       string `json:"id"`
	SenderId string `json:"sender_id"`
	Text     string `json:"text"`
}

type GETApiV1MessagesRequest struct {
	// Headers is a list of additional headers.
	Headers map[string]string
}

type GETApiV1MessagesResponse struct {
	Headers map[string][]string

	Body200 *MessagesResponseBody
}

func (cl *MessageService) GETApiV1Messages(
	ctx context.Context,
	request *GETApiV1MessagesRequest,
) (*GETApiV1MessagesResponse, error) {
	qos := cl.qos("GET /api/v1/messages")

	var response *GETApiV1MessagesResponse

	err := retry.OnError(ctx, qos.Retry, func(ctx context.Context) error {
		var err error

		response, err = cl.doGETApiV1Messages(ctx, qos, request)

		return err
	})
	if err != nil {
		return nil, fmt.Errorf("could not request: %w", err)
	}

	return response, nil
}

func (cl *MessageService) doGETApiV1Messages(
	ctx context.Context,
	qos config.QOS,
	request *GETApiV1MessagesRequest,
) (*GETApiV1MessagesResponse, error) {
	ctx, cancel := qos.Context(ctx)
	defer cancel()

	url := cl.baseURL.JoinPath("/api/v1/messages").String()

	var body io.Reader

	req, err := http.NewRequestWithContext(ctx, "GET", url, body)
	if err != nil {
		return nil, fmt.Errorf("could not prepare request: %w", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

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

