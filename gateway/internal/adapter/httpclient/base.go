package httpclient

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

	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

const UserNameHeader = "X-User-Name"

const requestTimeout = 5 * time.Second

type HttpDoer interface {
	do(ctx context.Context, call call) error
}

type BaseClient struct {
	httpClient *http.Client
	baseURL    string
}

var _ HttpDoer = (*BaseClient)(nil)

func NewBaseClient(baseURL string) *BaseClient {
	return &BaseClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

type call struct {
	method   string
	path     string
	query    url.Values
	username string
	body     any
	out      any
	statuses map[int]error
}

func (c *BaseClient) do(ctx context.Context, call call) error {
	request, err := c.newRequest(ctx, call)
	if err != nil {
		return err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return &HttpClientError{netErr: fmt.Errorf("call %s %s: %w", call.method, call.path, err)}
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		statusCode := response.StatusCode
		if mapped, ok := call.statuses[statusCode]; ok {
			return &HttpClientError{StatusCode: &statusCode, wrapped: mapped}
		}
		return unexpectedStatus(call, response)
	}

	if call.out == nil {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(call.out); err != nil {
		return &HttpClientError{wrapped: fmt.Errorf("decode %s %s response: %w", call.method, call.path, err)}
	}
	return nil
}

func (c *BaseClient) newRequest(ctx context.Context, call call) (*http.Request, error) {
	var body io.Reader
	if call.body != nil {
		encoded, err := json.Marshal(call.body)
		if err != nil {
			return nil, &HttpClientError{wrapped: fmt.Errorf("encode %s %s request: %w", call.method, call.path, err)}
		}
		body = bytes.NewReader(encoded)
	}

	target := c.baseURL + call.path
	if len(call.query) > 0 {
		target += "?" + call.query.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, call.method, target, body)
	if err != nil {
		return nil, &HttpClientError{wrapped: fmt.Errorf("build %s %s request: %w", call.method, call.path, err)}
	}
	if call.body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if call.username != "" {
		request.Header.Set(UserNameHeader, call.username)
	}
	return request, nil
}

func unexpectedStatus(call call, response *http.Response) error {
	statusCode := response.StatusCode
	clientErr := &HttpClientError{
		StatusCode: &statusCode,
		wrapped:    fmt.Errorf("call %s %s: unexpected status", call.method, call.path),
	}

	var payload struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err == nil && payload.Message != "" {
		clientErr.Message = &payload.Message
	}
	return clientErr
}

type pageResponse[T any] struct {
	Page          int `json:"page"`
	PageSize      int `json:"pageSize"`
	TotalElements int `json:"totalElements"`
	Items         []T `json:"items"`
}

func toPage[T, D any](response pageResponse[T], convert func(T) D) usecase.Page[D] {
	items := make([]D, 0, len(response.Items))
	for _, item := range response.Items {
		items = append(items, convert(item))
	}

	return usecase.Page[D]{
		Page:          response.Page,
		PageSize:      response.PageSize,
		TotalElements: response.TotalElements,
		Items:         items,
	}
}

func paging(page, size int) url.Values {
	return url.Values{
		"page": {fmt.Sprint(page)},
		"size": {fmt.Sprint(size)},
	}
}

type HttpClientError struct {
	StatusCode *int
	Message    *string
	netErr     error
	wrapped    error
}

func (e *HttpClientError) Error() string {
	if e.netErr != nil {
		return e.netErr.Error()
	}

	msg := "http error"
	if e.StatusCode != nil {
		msg += fmt.Sprintf(" with status %d", *e.StatusCode)
	}
	if e.Message != nil {
		msg += fmt.Sprintf(" (%s)", *e.Message)
	}
	if e.wrapped != nil {
		msg += ": " + e.wrapped.Error()
	}
	return msg
}

func (e *HttpClientError) Unwrap() error {
	if e.netErr != nil {
		return e.netErr
	}
	return e.wrapped
}
