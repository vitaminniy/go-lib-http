package messageservice

import (
	_ "bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

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
