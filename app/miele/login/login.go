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
}

// NewManager builds a Manager with default Miele API clients. Tests can
// substitute clients via the exported fields.
func NewManager() *Manager {
	return &Manager{
		login:   NewClient(),
		devices: api.NewClient(),
		now:     time.Now,
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
		validUntil = m.now().Add(time.Hour)
	}
	m.SetToken(&Token{
		AccessToken:  tc.Access,
		RefreshToken: tc.Refresh,
		TokenType:    "Bearer",
		ExpiresAt:    validUntil,
	})
	logger.Info("Recovered token from config")
}

// Login obtains a fresh token, refreshing when possible and falling back to
// the full username/password flow when not.
//
// Mirrors lib/miele/login/login.ts:
//  1. If we have a token, try fetching devices with it.
//  2. If that fails OR we are within 24h of expiry, try the refresh-token grant.
//  3. If refresh fails OR we never had a token, perform the code+token exchange.
//
// On success the token is stored in memory and (when config.miele.persistToken
// is true) written to the config file.
func (m *Manager) Login(ctx context.Context) (*Token, error) {
	cfg := config.Get().Miele
	now := m.now()

	connected := m.assertConnection(ctx)
	current := m.Current()

	if current != nil && current.RefreshToken != "" && (!connected || NeedsRefresh(current, now)) {
		res, err := m.login.RefreshToken(ctx, cfg, current.RefreshToken)
		if err != nil {
			logger.Error("Token refresh failed. Falling back to full login.", "error", err)
			connected = false
		} else {
			t := Convert(res, m.now())
			m.SetToken(&t)
			connected = true
		}
	}

	if !connected || m.Current() == nil {
		code, err := m.login.FetchCode(ctx, cfg)
		if err != nil {
			return nil, err
		}
		res, err := m.login.FetchToken(ctx, cfg, code)
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
		config.PersistToken(config.TokenConfig{
			Access:     final.AccessToken,
			Refresh:    final.RefreshToken,
			ValidUntil: final.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}

	if m.onRefresh != nil {
		m.onRefresh(m.now(), final.ExpiresAt)
	}

	logger.Info("Login successful")
	return final, nil
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
