package login

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// TokenResult is the wire shape returned by Miele's token endpoints (legacy
// /thirdparty/token and the OAuth2 provider's openid-connect/token).
type TokenResult struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in,omitempty"`
}

// Token is the in-process token, with absolute expiries derived from
// ExpiresIn and RefreshExpiresIn at the time the token was minted.
type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    time.Time
	// RefreshExpiresAt is when the refresh token expires and a new login is
	// required. Zero when the provider did not report it.
	RefreshExpiresAt time.Time
}

// Convert turns a freshly-fetched TokenResult into a Token using now as the
// reference time for expiry computation.
func Convert(r TokenResult, now time.Time) Token {
	t := Token{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		TokenType:    r.TokenType,
		ExpiresAt:    now.Add(time.Duration(r.ExpiresIn) * time.Second),
	}
	if r.RefreshExpiresIn > 0 {
		t.RefreshExpiresAt = now.Add(time.Duration(r.RefreshExpiresIn) * time.Second)
	}
	return t
}

// legacyTokenRe matches tokens issued by the legacy provider: a two-letter
// country prefix, an underscore and 32 hex digits (36 chars, e.g. "DE_…").
var legacyTokenRe = regexp.MustCompile(`^[A-Z]{2}_[0-9a-fA-F]{32}$`)

// IsLegacyToken reports whether tok was issued by the legacy OAuth provider.
// Tokens of the new OAuth2 provider are much longer signed JWTs.
func IsLegacyToken(tok string) bool {
	return legacyTokenRe.MatchString(tok)
}

const (
	// legacyRefreshThreshold matches lib/miele/login/login.ts:needsRefresh.
	legacyRefreshThreshold = 24 * time.Hour
	// oauth2RefreshThreshold applies to the short-lived tokens of the new
	// provider; the refresh check runs once per minute.
	oauth2RefreshThreshold = 5 * time.Minute
)

// NeedsRefresh returns true when the token is within its refresh threshold of
// expiring (or already expired): one day for legacy tokens, five minutes for
// tokens of the new OAuth2 provider.
func NeedsRefresh(t *Token, now time.Time) bool {
	if t == nil {
		return false
	}
	threshold := oauth2RefreshThreshold
	if IsLegacyToken(t.AccessToken) {
		threshold = legacyRefreshThreshold
	}
	return !t.ExpiresAt.After(now.Add(threshold))
}

// jwtExpiry returns the `exp` claim of a JWT without verifying its signature.
// ok is false when tok is not a JWT or carries no exp claim.
func jwtExpiry(tok string) (exp time.Time, ok bool) {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}
