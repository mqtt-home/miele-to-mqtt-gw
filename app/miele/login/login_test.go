package login

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
	"github.com/mqtt-home/miele-to-mqtt-gw/miele/api"
)

// stubMiele wires three pluggable HTTP handlers (oauth/auth, /thirdparty/token,
// /v1/devices/) into a single httptest server so tests can simulate the whole
// flow without hand-wiring three servers.
func stubMiele(t *testing.T, h struct {
	code    http.HandlerFunc
	token   http.HandlerFunc
	devices http.HandlerFunc
}) (*httptest.Server, *Client, *api.Client) {
	t.Helper()
	mux := http.NewServeMux()
	if h.code != nil {
		mux.HandleFunc("/oauth/auth", h.code)
	}
	if h.token != nil {
		mux.HandleFunc("/thirdparty/token", h.token)
	}
	if h.devices != nil {
		mux.HandleFunc("/v1/devices/", h.devices)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	lc := &Client{
		HTTP: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		BaseURL: srv.URL,
	}
	apiC := &api.Client{HTTP: srv.Client(), BaseURL: srv.URL}
	return srv, lc, apiC
}

func TestFetchCode_Success(t *testing.T) {
	_, lc, _ := stubMiele(t, struct {
		code    http.HandlerFunc
		token   http.HandlerFunc
		devices http.HandlerFunc
	}{
		code: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
				t.Errorf("Content-Type = %q", got)
			}
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			if form.Get("email") != "alice" {
				t.Errorf("email = %q", form.Get("email"))
			}
			if form.Get("password") != "pw" {
				t.Errorf("password = %q", form.Get("password"))
			}
			if form.Get("client_id") != "cid" {
				t.Errorf("client_id = %q", form.Get("client_id"))
			}
			if form.Get("vgInformationSelector") != "de-DE" {
				t.Errorf("country = %q", form.Get("vgInformationSelector"))
			}
			w.Header().Set("Location", "/v1/?code=THECODE&state=login")
			w.WriteHeader(http.StatusFound)
		},
	})

	cfg := config.MieleConfig{Username: "alice", Password: "pw", ClientID: "cid", CountryCode: "de-DE"}
	code, err := lc.FetchCode(context.Background(), cfg)
	if err != nil {
		t.Fatalf("FetchCode: %v", err)
	}
	if code != "THECODE" {
		t.Errorf("code = %q, want THECODE", code)
	}
}

func TestFetchCode_MissingLocation(t *testing.T) {
	_, lc, _ := stubMiele(t, struct {
		code    http.HandlerFunc
		token   http.HandlerFunc
		devices http.HandlerFunc
	}{
		code: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusFound)
		},
	})
	_, err := lc.FetchCode(context.Background(), config.MieleConfig{})
	if err == nil {
		t.Fatal("expected error for missing Location")
	}
}

func TestFetchToken_Success(t *testing.T) {
	_, lc, _ := stubMiele(t, struct {
		code    http.HandlerFunc
		token   http.HandlerFunc
		devices http.HandlerFunc
	}{
		token: func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			if form.Get("grant_type") != "authorization_code" {
				t.Errorf("grant_type = %q", form.Get("grant_type"))
			}
			if form.Get("code") != "thecode" {
				t.Errorf("code = %q", form.Get("code"))
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"a","refresh_token":"r","token_type":"Bearer","expires_in":3600}`))
		},
	})

	res, err := lc.FetchToken(context.Background(), config.MieleConfig{ClientID: "cid", ClientSecret: "csec"}, "thecode")
	if err != nil {
		t.Fatalf("FetchToken: %v", err)
	}
	if res.AccessToken != "a" || res.RefreshToken != "r" || res.ExpiresIn != 3600 {
		t.Errorf("token = %+v", res)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	_, lc, _ := stubMiele(t, struct {
		code    http.HandlerFunc
		token   http.HandlerFunc
		devices http.HandlerFunc
	}{
		token: func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			if form.Get("grant_type") != "refresh_token" {
				t.Errorf("grant_type = %q", form.Get("grant_type"))
			}
			if form.Get("refresh_token") != "RT" {
				t.Errorf("refresh_token = %q", form.Get("refresh_token"))
			}
			w.Write([]byte(`{"access_token":"a2","refresh_token":"r2","token_type":"Bearer","expires_in":7200}`))
		},
	})
	res, err := lc.RefreshToken(context.Background(), config.MieleConfig{ClientID: "cid", ClientSecret: "csec"}, "RT")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if res.AccessToken != "a2" || res.RefreshToken != "r2" {
		t.Errorf("token = %+v", res)
	}
}

func TestConvert(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tok := Convert(TokenResult{AccessToken: "a", RefreshToken: "r", TokenType: "Bearer", ExpiresIn: 3600}, now)
	want := now.Add(time.Hour)
	if !tok.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", tok.ExpiresAt, want)
	}
}

func TestNeedsRefresh(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		expiry time.Duration
		want   bool
	}{
		{"expires in 25h", 25 * time.Hour, false},
		{"expires in 24h exactly", 24 * time.Hour, true},
		{"expires in 23h", 23 * time.Hour, true},
		{"already expired", -time.Hour, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tok := &Token{ExpiresAt: now.Add(c.expiry)}
			if got := NeedsRefresh(tok, now); got != c.want {
				t.Errorf("NeedsRefresh = %v, want %v", got, c.want)
			}
		})
	}
	if NeedsRefresh(nil, now) {
		t.Error("nil token should not need refresh")
	}
}

func TestRecoverFromConfig_WithValidUntil(t *testing.T) {
	m := NewManager()
	m.SetNowFunc(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) })
	m.RecoverFromConfig(config.Config{
		Miele: config.MieleConfig{
			Token: &config.TokenConfig{
				Access:     "a",
				Refresh:    "r",
				ValidUntil: "2030-01-01T00:00:00Z",
			},
		},
	})
	t1 := m.Current()
	if t1 == nil || t1.AccessToken != "a" {
		t.Fatalf("token not recovered: %+v", t1)
	}
	want, _ := time.Parse(time.RFC3339, "2030-01-01T00:00:00Z")
	if !t1.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", t1.ExpiresAt, want)
	}
}

func TestRecoverFromConfig_NoValidUntilDefaultsTo1h(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := NewManager()
	m.SetNowFunc(func() time.Time { return now })
	m.RecoverFromConfig(config.Config{
		Miele: config.MieleConfig{
			Token: &config.TokenConfig{Access: "a", Refresh: "r"},
		},
	})
	tok := m.Current()
	if tok == nil {
		t.Fatal("token nil")
	}
	want := now.Add(time.Hour)
	if !tok.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", tok.ExpiresAt, want)
	}
}

func TestRecoverFromConfig_NoToken(t *testing.T) {
	m := NewManager()
	m.RecoverFromConfig(config.Config{})
	if m.Current() != nil {
		t.Error("expected no token")
	}
}

func TestLogin_FullFlow_NoExistingToken(t *testing.T) {
	srv, lc, apiC := stubMiele(t, struct {
		code    http.HandlerFunc
		token   http.HandlerFunc
		devices http.HandlerFunc
	}{
		code: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "/v1/?code=C")
			w.WriteHeader(http.StatusFound)
		},
		token: func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(`{"access_token":"a","refresh_token":"r","token_type":"Bearer","expires_in":3600}`))
		},
	})
	_ = srv

	// Use a config that turns persistToken off so we don't write to disk.
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	os.WriteFile(p, []byte(`{
        "mqtt": {"url": "tcp://x", "topic": "miele"},
        "miele": {"client-id": "cid", "client-secret": "csec", "username": "u", "password": "p", "persistToken": false}
    }`), 0o600)
	if _, err := config.LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	m := NewManager()
	m.SetClients(lc, apiC)
	m.SetNowFunc(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) })

	tok, err := m.Login(context.Background())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok.AccessToken != "a" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
}

func TestLogin_RefreshPath(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	devicesCalls := 0
	tokenCalls := 0
	codeCalls := 0
	_, lc, apiC := stubMiele(t, struct {
		code    http.HandlerFunc
		token   http.HandlerFunc
		devices http.HandlerFunc
	}{
		code: func(w http.ResponseWriter, _ *http.Request) {
			codeCalls++
			w.Header().Set("Location", "/v1/?code=NEW")
			w.WriteHeader(http.StatusFound)
		},
		token: func(w http.ResponseWriter, r *http.Request) {
			tokenCalls++
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			if form.Get("grant_type") != "refresh_token" {
				t.Errorf("expected refresh, got %q", form.Get("grant_type"))
			}
			w.Write([]byte(`{"access_token":"new-a","refresh_token":"new-r","token_type":"Bearer","expires_in":3600}`))
		},
		devices: func(w http.ResponseWriter, _ *http.Request) {
			devicesCalls++
			w.Write([]byte(`{}`))
		},
	})

	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	os.WriteFile(p, []byte(`{
        "mqtt": {"url": "tcp://x", "topic": "miele"},
        "miele": {"client-id": "cid", "client-secret": "csec", "persistToken": false}
    }`), 0o600)
	if _, err := config.LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	m := NewManager()
	m.SetClients(lc, apiC)
	m.SetNowFunc(func() time.Time { return now })
	// Token that expires in 12h => needs refresh.
	m.SetToken(&Token{AccessToken: "old-a", RefreshToken: "old-r", ExpiresAt: now.Add(12 * time.Hour)})

	tok, err := m.Login(context.Background())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok.AccessToken != "new-a" {
		t.Errorf("AccessToken = %q, want new-a", tok.AccessToken)
	}
	if tokenCalls != 1 {
		t.Errorf("tokenCalls = %d, want 1", tokenCalls)
	}
	if codeCalls != 0 {
		t.Errorf("codeCalls = %d, want 0 (no full login expected)", codeCalls)
	}
	if devicesCalls != 1 {
		t.Errorf("devicesCalls = %d, want 1 (assert connection)", devicesCalls)
	}
}

func TestLogin_RefreshFailure_FallsBackToFullLogin(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tokenCalls := 0
	codeCalls := 0
	_, lc, apiC := stubMiele(t, struct {
		code    http.HandlerFunc
		token   http.HandlerFunc
		devices http.HandlerFunc
	}{
		code: func(w http.ResponseWriter, _ *http.Request) {
			codeCalls++
			w.Header().Set("Location", "/v1/?code=NEW")
			w.WriteHeader(http.StatusFound)
		},
		token: func(w http.ResponseWriter, r *http.Request) {
			tokenCalls++
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			if form.Get("grant_type") == "refresh_token" {
				http.Error(w, "refused", http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`{"access_token":"new-a","refresh_token":"new-r","token_type":"Bearer","expires_in":3600}`))
		},
		devices: func(w http.ResponseWriter, _ *http.Request) {
			// Pretend we're disconnected so we go down the refresh path.
			http.Error(w, "no", http.StatusUnauthorized)
		},
	})

	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	os.WriteFile(p, []byte(`{
        "mqtt": {"url": "tcp://x", "topic": "miele"},
        "miele": {"client-id": "cid", "client-secret": "csec", "persistToken": false}
    }`), 0o600)
	if _, err := config.LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	m := NewManager()
	m.SetClients(lc, apiC)
	m.SetNowFunc(func() time.Time { return now })
	m.SetToken(&Token{AccessToken: "old-a", RefreshToken: "old-r", ExpiresAt: now.Add(12 * time.Hour)})

	tok, err := m.Login(context.Background())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok.AccessToken != "new-a" {
		t.Errorf("AccessToken = %q, want new-a", tok.AccessToken)
	}
	if codeCalls != 1 {
		t.Errorf("codeCalls = %d, want 1 (fallback full login)", codeCalls)
	}
	if tokenCalls != 2 {
		t.Errorf("tokenCalls = %d, want 2 (refresh + full)", tokenCalls)
	}
}

func TestLogin_PersistsTokenWhenEnabled(t *testing.T) {
	_, lc, apiC := stubMiele(t, struct {
		code    http.HandlerFunc
		token   http.HandlerFunc
		devices http.HandlerFunc
	}{
		code: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "/v1/?code=C")
			w.WriteHeader(http.StatusFound)
		},
		token: func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(`{"access_token":"a","refresh_token":"r","token_type":"Bearer","expires_in":3600}`))
		},
	})

	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	os.WriteFile(p, []byte(`{
        "mqtt": {"url": "tcp://x", "topic": "miele"},
        "miele": {"client-id": "cid", "client-secret": "csec", "username": "u", "password": "p"}
    }`), 0o600)
	if _, err := config.LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	m := NewManager()
	m.SetClients(lc, apiC)
	m.SetNowFunc(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) })

	if _, err := m.Login(context.Background()); err != nil {
		t.Fatalf("Login: %v", err)
	}

	out, _ := os.ReadFile(p)
	if !strings.Contains(string(out), `"access": "a"`) && !strings.Contains(string(out), `"access":"a"`) {
		t.Errorf("token not persisted: %s", string(out))
	}
}
