package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// BaseURL is the Miele cloud API root used by all REST calls.
	BaseURL = "https://api.mcs3.miele.com"

	// DevicesPath is the REST endpoint listing all devices for the
	// authenticated account.
	DevicesPath = "/v1/devices/"

	// PingPath is a reachability probe (the same one used by the TS
	// implementation).
	PingPath = "/thirdparty/login/"
)

// Client is a thin HTTP wrapper around the Miele REST API. It is safe for
// concurrent use.
type Client struct {
	HTTP    *http.Client
	BaseURL string
}

// NewClient builds a client with a sensible default timeout. 60s matches the
// polling interval — slow Miele responses are common and the TypeScript
// version (axios with no timeout) effectively waited forever.
func NewClient() *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 60 * time.Second},
		BaseURL: BaseURL,
	}
}

// FetchDevices fetches and parses the device list. The response is an object
// keyed by device serial; we surface it as a slice so downstream code does not
// have to care about map iteration order.
func (c *Client) FetchDevices(ctx context.Context, accessToken string) ([]Device, error) {
	if accessToken == "" {
		return nil, errors.New("fetch devices: empty access token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+DevicesPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch devices: status %d: %s", resp.StatusCode, string(body))
	}

	return parseDevices(resp.Body)
}

func parseDevices(r io.Reader) ([]Device, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	// Use a Decoder with UseNumber? Not needed — we keep raw bytes per device.
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse devices: %w", err)
	}

	out := make([]Device, 0, len(raw))
	for id, payload := range raw {
		out = append(out, Device{ID: id, Data: payload})
	}
	return out, nil
}

// Ping returns true when the Miele cloud is reachable; it matches the
// existing TS `ping()` helper and is used by the connection-check loop.
func (c *Client) Ping(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+PingPath, nil)
	if err != nil {
		return false
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}
