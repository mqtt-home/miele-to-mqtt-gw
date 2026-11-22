package login

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
	"github.com/mqtt-home/miele-to-mqtt-gw/miele/api"
	"github.com/philipparndt/go-logger"
)

// Manager owns the current token, performs login/refresh, and persists tokens
// back to the config file when enabled. It is safe for concurrent use.
type Manager struct {
	mu        sync.RWMutex
	tok       *Token
	login     *Client
	devices   *api.Client
	now       func() time.Time
	onRefresh func(now, expiresAt time.Time)

	onAuthRequired func()
	authorize      func(ctx context.Context, cfg config.MieleConfig, c *Client) (TokenResult, error)
}

// NewManager builds a Manager with default Miele API clients. Tests can
// substitute clients via the exported fields.
func NewManager() *Manager {
	return &Manager{
		login:     NewClient(),
		devices:   api.NewClient(),
		now:       time.Now,
		authorize: Authorize,
	}
}

// SetClients lets tests inject stubbed HTTP clients.
func (m *Manager) SetClients(login *Client, devices *api.Client) {
	m.login = login
	m.devices = devices
}

// SetNowFunc lets tests pin the clock.
func (m *Manager) SetNowFunc(now func() time.Time) {
	m.now = now
}

// SetOnRefresh registers a callback invoked after every successful Login
// (whether via refresh-token or full code+token). It is used by the metrics
// layer to record refresh events without introducing an import dependency.
func (m *Manager) SetOnRefresh(fn func(now, expiresAt time.Time)) {
	m.onRefresh = fn
}

// SetOnAuthRequired registers a callback invoked when a browser login becomes
// necessary, before the login page is started.
func (m *Manager) SetOnAuthRequired(fn func()) {
	m.onAuthRequired = fn
}

// SetAuthorizer lets tests replace the interactive browser login.
func (m *Manager) SetAuthorizer(fn func(ctx context.Context, cfg config.MieleConfig, c *Client) (TokenResult, error)) {
	m.authorize = fn
}

// Current returns the in-memory token, or nil if none.
func (m *Manager) Current() *Token {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.tok == nil {
		return nil
	}
	t := *m.tok
	return &t
}

// CurrentAccessToken returns the current bearer string or "".
func (m *Manager) CurrentAccessToken() string {
	t := m.Current()
	if t == nil {
		return ""
	}
	return t.AccessToken
}

// SetToken installs a token without touching the network. Used to bootstrap
// from config.miele.token.
func (m *Manager) SetToken(t *Token) {
	m.mu.Lock()
	m.tok = t
	m.mu.Unlock()
}

// RecoverFromConfig installs a token from a config block when present. If
// validUntil is missing or unparseable, expiry defaults to now+1h to match
// lib/config/config.ts:recoverToken.
func (m *Manager) RecoverFromConfig(c config.Config) {
	if c.Miele.Token == nil {
		return
	}
	tc := c.Miele.Token
	var validUntil time.Time
	if tc.ValidUntil != "" {
		if parsed, err := time.Parse(time.RFC3339, tc.ValidUntil); err == nil {
			validUntil = parsed
		}
	}
	if validUntil.IsZero() {
		if exp, ok := jwtExpiry(tc.Access); ok {
			validUntil = exp
		}
	}
	if validUntil.IsZero() {
		validUntil = m.now().Add(time.Hour)
	}
	var refreshValidUntil time.Time
	if tc.RefreshValidUntil != "" {
		if parsed, err := time.Parse(time.RFC3339, tc.RefreshValidUntil); err == nil {
			refreshValidUntil = parsed
		}
	}
	m.SetToken(&Token{
		AccessToken:      tc.Access,
		RefreshToken:     tc.Refresh,
		TokenType:        "Bearer",
		ExpiresAt:        validUntil,
		RefreshExpiresAt: refreshValidUntil,
	})
	logger.Info("Recovered token from config")
}

// Login obtains a fresh token, refreshing when possible and falling back to
// a full login when not.
//
// Mirrors lib/miele/login/login.ts:
//  1. If we have a token, try fetching devices with it.
//  2. If that fails OR the token is about to expire, try the refresh-token
//     grant. The stored token's format selects the provider: legacy tokens
//     ("DE_" + 32 hex) are refreshed at the legacy endpoint, all others at
//     the OAuth2 provider.
//  3. If refresh fails OR we never had a token, perform the full login
//     selected by miele.auth-type: the legacy username/password code+token
//     exchange, or the browser login of the OAuth2 provider, which blocks
//     until the user has logged in.
//
// On success the token is stored in memory and (when config.miele.persistToken
// is true) written to the config file.
func (m *Manager) Login(ctx context.Context) (*Token, error) {
	cfg := config.Get().Miele
	authType := cfg.ResolvedAuthType()
	now := m.now()

	if cfg.AuthType == config.AuthTypeDevice {
		if t := m.Current(); t != nil && IsLegacyToken(t.AccessToken) {
			logger.Info("Discarding legacy token: auth-type is per-device, a browser login is required")
			m.SetToken(nil)
		}
	}

	connected := m.assertConnection(ctx)
	current := m.Current()

	if current != nil && current.RefreshToken != "" && (!connected || NeedsRefresh(current, now)) {
		res, err := m.refresh(ctx, cfg, current)
		if err != nil {
			logger.Error("Token refresh failed. Falling back to full login.", "error", err)
			connected = false
		} else {
			t := Convert(res, m.now())
			// RFC 6749 §6: a refresh response may omit refresh_token, in
			// which case the previous one stays valid.
			if t.RefreshToken == "" {
				t.RefreshToken = current.RefreshToken
				t.RefreshExpiresAt = current.RefreshExpiresAt
			}
			m.SetToken(&t)
			connected = true
		}
	}

	if !connected || m.Current() == nil {
		var res TokenResult
		var err error
		if authType == config.AuthTypeLegacy {
			res, err = m.legacyLogin(ctx, cfg)
		} else {
			res, err = m.browserLogin(ctx, cfg)
		}
		if err != nil {
			return nil, err
		}
		t := Convert(res, m.now())
		m.SetToken(&t)
	}

	final := m.Current()
	if final == nil {
		return nil, errors.New("login: token is nil after login flow")
	}

	if cfg.PersistToken {
		tc := config.TokenConfig{
			Access:     final.AccessToken,
			Refresh:    final.RefreshToken,
			ValidUntil: final.ExpiresAt.UTC().Format(time.RFC3339),
		}
		if !final.RefreshExpiresAt.IsZero() {
			tc.RefreshValidUntil = final.RefreshExpiresAt.UTC().Format(time.RFC3339)
		}
		config.PersistToken(tc)
	}

	if m.onRefresh != nil {
		m.onRefresh(m.now(), final.ExpiresAt)
	}

	logger.Info("Login successful")
	return final, nil
}

// refresh runs the refresh-token grant at the provider that issued t.
func (m *Manager) refresh(ctx context.Context, cfg config.MieleConfig, t *Token) (TokenResult, error) {
	if IsLegacyToken(t.AccessToken) {
		return m.login.RefreshToken(ctx, cfg, t.RefreshToken)
	}
	res, err := m.login.RefreshOAuth2(ctx, cfg, t.RefreshToken)
	if err == nil {
		logger.Debug("Refreshed OAuth2 token", "expires_in", res.ExpiresIn, "refresh_expires_in", res.RefreshExpiresIn)
	}
	return res, err
}

// legacyLogin performs the username/password code+token exchange.
func (m *Manager) legacyLogin(ctx context.Context, cfg config.MieleConfig) (TokenResult, error) {
	code, err := m.login.FetchCode(ctx, cfg)
	if err != nil {
		return TokenResult{}, err
	}
	return m.login.FetchToken(ctx, cfg, code)
}

// browserLogin runs the interactive login of the OAuth2 provider.
func (m *Manager) browserLogin(ctx context.Context, cfg config.MieleConfig) (TokenResult, error) {
	if m.onAuthRequired != nil {
		m.onAuthRequired()
	}
	res, err := m.authorize(ctx, cfg, m.login)
	if err != nil {
		return TokenResult{}, err
	}
	logger.Info("Received OAuth2 token", "expires_in", res.ExpiresIn, "refresh_expires_in", res.RefreshExpiresIn)
	return res, nil
}

// assertConnection mirrors lib/miele/login/login.ts:assertConnection: a
// successful FetchDevices call indicates we still have a working token.
func (m *Manager) assertConnection(ctx context.Context) bool {
	t := m.Current()
	if t == nil || t.AccessToken == "" {
		return false
	}
	if _, err := m.devices.FetchDevices(ctx, t.AccessToken); err != nil {
		logger.Debug("Connection assertion failed", "error", err)
		return false
	}
	return true
}
