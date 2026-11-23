package login

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
)

// pkce holds the PKCE verifier/challenge pair (RFC 7636, S256) of one
// pending browser login.
type pkce struct {
	Verifier  string
	Challenge string
}

func newPKCE() (pkce, error) {
	v, err := randomString(32)
	if err != nil {
		return pkce{}, err
	}
	sum := sha256.Sum256([]byte(v))
	return pkce{Verifier: v, Challenge: base64.RawURLEncoding.EncodeToString(sum[:])}, nil
}

// randomString returns n random bytes, base64url encoded.
func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// AuthorizeURL builds the URL the user opens in a browser to log in at the
// OAuth2 provider and grant access to selected appliances.
func (c *Client) AuthorizeURL(cfg config.MieleConfig, redirectURI, state, codeChallenge string) string {
	q := url.Values{}
	q.Set("client_id", cfg.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", cfg.Scope)
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	// Miele documents scopes as %20-separated; Encode would use "+". Literal
	// "+" characters are already escaped as %2B, so the replacement is safe.
	return c.AuthBaseURL + oauth2AuthPath + "?" + strings.ReplaceAll(q.Encode(), "+", "%20")
}

// ExchangeCode exchanges an authorization code from the browser login for
// tokens. redirectURI must be identical to the one used in AuthorizeURL.
func (c *Client) ExchangeCode(ctx context.Context, cfg config.MieleConfig, code, redirectURI, codeVerifier string) (TokenResult, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", codeVerifier)
	return c.postToken(ctx, c.AuthBaseURL+oauth2TokenPath, form)
}

// RefreshOAuth2 exchanges a refresh_token of the OAuth2 provider for new
// tokens. The provider may rotate the refresh token, so callers must store the
// returned one.
func (c *Client) RefreshOAuth2(ctx context.Context, cfg config.MieleConfig, refreshToken string) (TokenResult, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")
	return c.postToken(ctx, c.AuthBaseURL+oauth2TokenPath, form)
}
