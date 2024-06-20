package messageservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// This is needed to have bytes imported when non-body requests are generated.
var _ = bytes.Buffer{}

// NewMessageService creates a new MessageService http client.
func NewMessageService(baseurl string) (*MessageService, error) {
	parsed, err := url.Parse(baseurl)
	if err != nil {
		return nil, fmt.Errorf("could not parse base url: %w", err)
	}

	return &MessageService{
		baseURL:    parsed,
		httpClient: http.DefaultClient,
	}, nil
}

type MessageService struct {
	baseURL    *url.URL
	httpClient *http.Client
}

type MessageRequestBody struct {
	SenderId string `json:"sender_id"`
	Text     string `json:"text"`
	Meta     string `json:"meta,omitempty"`
}

type MessageResponseBody struct {
	Id   string `json:"id"`
	Meta string `json:"meta,omitempty"`
}

type POSTApiV1MessageRequest struct {
	// Headers is a list of additional headers.
	Headers map[string]string

	// Body is a request body.
	Body *MessageRequestBody
}

type POSTApiV1MessageResponse struct {
	Headers map[string][]string

	Body201 *MessageResponseBody
}

func (cl *MessageService) POSTApiV1Message(
	ctx context.Context,
	request *POSTApiV1MessageRequest,
) (*POSTApiV1MessageResponse, error) {
	url := cl.baseURL.JoinPath("/api/v1/message").String()

	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(&request.Body); err != nil {
		return nil, fmt.Errorf("could not encode request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
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

	response := &POSTApiV1MessageResponse{
		Headers: resp.Header,
	}

	if resp.StatusCode == 201 {
		var body MessageResponseBody
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return nil, fmt.Errorf("could not decode response [%d]: %w", resp.StatusCode, err)
		}

		response.Body201 = &body

		return response, nil
	}

	return nil, fmt.Errorf("unhandled response code: %d", resp.StatusCode)
}
