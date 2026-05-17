package login

import "time"

// TokenResult is the wire shape returned by Miele's /thirdparty/token.
type TokenResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Token is the in-process token, with an absolute expiry derived from
// ExpiresIn at the time the token was minted.
type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    time.Time
}

// Convert turns a freshly-fetched TokenResult into a Token using now as the
// reference time for expiry computation.
func Convert(r TokenResult, now time.Time) Token {
	return Token{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		TokenType:    r.TokenType,
		ExpiresAt:    now.Add(time.Duration(r.ExpiresIn) * time.Second),
	}
}

// NeedsRefresh returns true when the token is within one day of expiring (or
// already expired). Matches lib/miele/login/login.ts:needsRefresh.
func NeedsRefresh(t *Token, now time.Time) bool {
	if t == nil {
		return false
	}
	return !t.ExpiresAt.After(now.Add(24 * time.Hour))
}
