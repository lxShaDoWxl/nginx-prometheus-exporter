package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// NginxClient allows you to fetch NGINX metrics from the status page.
type NginxClient struct {
	apiEndpoint string
	httpClient  *http.Client
}

// Status represents NGINX metrics.
type Status struct {
	Connections struct {
		Accepted int64 `json:"accepted"`
		Active   int64 `json:"active"`
		Idle     int64 `json:"idle"`
		Closed   int64 `json:"closed"`
	} `json:"connections"`
	Requests struct {
		Total int64 `json:"total"`
	} `json:"requests"`
	Applications map[string]struct {
		Processes struct {
			Running  int `json:"running"`
			Starting int `json:"starting"`
			Idle     int `json:"idle"`
		} `json:"processes"`
		Requests struct {
			Active int `json:"active"`
		} `json:"requests"`
	} `json:"applications"`
}

// NewNginxClient creates an NginxClient.
func NewNginxClient(httpClient *http.Client, apiEndpoint string) (*NginxClient, error) {
	client := &NginxClient{
		apiEndpoint: apiEndpoint,
		httpClient:  httpClient,
	}

	_, err := client.GetStatus()
	return client, err
}

// GetStatus fetches the metrics.
func (client *NginxClient) GetStatus() (*Status, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.apiEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create a get request: %w", err)
	}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get %v: %w", client.apiEndpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected %v response, got %v", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response body: %w", err)
	}
	status := &Status{}

	err = json.Unmarshal(body, status)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response body %q: %w", string(body), err)
	}

	return status, nil
}
