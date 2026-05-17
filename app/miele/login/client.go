package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
)

const (
	// BaseURL is the Miele cloud root; tests inject their own via Client.BaseURL.
	BaseURL = "https://api.mcs3.miele.com"

	codePath  = "/oauth/auth"
	tokenPath = "/thirdparty/token"
)

// Client performs the Miele OAuth2 flow. It is safe for concurrent use; the
// HTTP client never follows redirects so we can read the Location header
// directly from the 302 response.
type Client struct {
	HTTP    *http.Client
	BaseURL string
}

// NewClient returns a Client with a sensible timeout and a redirect policy
// that surfaces 302 to the caller (the code-fetch step depends on this).
func NewClient() *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		BaseURL: BaseURL,
	}
}

// FetchCode performs the form post against /oauth/auth and extracts the
// `code` query parameter from the resulting 302 Location header. Mirrors
// lib/miele/login/code.ts.
func (c *Client) FetchCode(ctx context.Context, cfg config.MieleConfig) (string, error) {
	form := url.Values{}
	form.Set("email", cfg.Username)
	form.Set("password", cfg.Password)
	form.Set("redirect_uri", "/v1/")
	form.Set("state", "login")
	form.Set("response_type", "code")
	form.Set("client_id", cfg.ClientID)
	form.Set("vgInformationSelector", cfg.CountryCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+codePath, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("fetch code: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", errors.New("fetch code: Location header missing")
	}
	u, err := url.Parse(loc)
	if err != nil {
		return "", fmt.Errorf("fetch code: parse Location: %w", err)
	}
	code := u.Query().Get("code")
	if code == "" {
		return "", errors.New("fetch code: code missing in Location")
	}
	return code, nil
}

// FetchToken exchanges an authorization code for tokens.
func (c *Client) FetchToken(ctx context.Context, cfg config.MieleConfig, code string) (TokenResult, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", "/v1/devices")
	form.Set("grant_type", "authorization_code")
	form.Set("state", "token")
	return c.postToken(ctx, form)
}

// RefreshToken exchanges a refresh_token for a new access_token.
func (c *Client) RefreshToken(ctx context.Context, cfg config.MieleConfig, refreshToken string) (TokenResult, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")
	return c.postToken(ctx, form)
}

func (c *Client) postToken(ctx context.Context, form url.Values) (TokenResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+tokenPath, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return TokenResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return TokenResult{}, fmt.Errorf("token endpoint: status %d: %s", resp.StatusCode, string(body))
	}

	var out TokenResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return TokenResult{}, fmt.Errorf("token endpoint: decode: %w", err)
	}
	return out, nil
}
