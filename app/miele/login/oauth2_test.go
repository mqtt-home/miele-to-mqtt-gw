package login

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
	"github.com/mqtt-home/miele-to-mqtt-gw/miele/api"
)

// legacyOld is a token in the legacy provider's format.
const legacyOld = "DE_0123456789abcdef0123456789abcdef"

// fakeJWT builds an unsigned JWT-shaped token with the given exp claim.
func fakeJWT(exp int64) string {
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." +
		enc([]byte(fmt.Sprintf(`{"exp":%d,"sub":"user"}`, exp))) + ".c2lnbmF0dXJl"
}

// tokenRecorder records the forms posted to a token endpoint.
type tokenRecorder struct {
	mu    sync.Mutex
	forms []url.Values
	body  string
}

func (rec *tokenRecorder) handler(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	form, _ := url.ParseQuery(string(b))
	rec.mu.Lock()
	rec.forms = append(rec.forms, form)
	body := rec.body
	rec.mu.Unlock()
	if body == "" {
		http.Error(w, "refused", http.StatusUnauthorized)
		return
	}
	w.Write([]byte(body))
}

func (rec *tokenRecorder) calls() []url.Values {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return append([]url.Values(nil), rec.forms...)
}

// stubBoth serves the legacy token endpoint, the OAuth2 token endpoint and
// /v1/devices/ from one httptest server.
func stubBoth(t *testing.T, legacy, oauth2 *tokenRecorder, devices http.HandlerFunc) (*Client, *api.Client) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/thirdparty/token", legacy.handler)
	mux.HandleFunc("/oidc/token", oauth2.handler)
	if devices == nil {
		devices = func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "no", http.StatusUnauthorized) }
	}
	mux.HandleFunc("/v1/devices/", devices)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	lc := NewClient()
	lc.BaseURL = srv.URL
	lc.AuthBaseURL = srv.URL + "/oidc"
	return lc, &api.Client{HTTP: srv.Client(), BaseURL: srv.URL}
}

func loadTestConfig(t *testing.T, miele string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	body := `{"mqtt": {"url": "tcp://x", "topic": "miele"}, "miele": ` + miele + `}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
}

func TestIsLegacyToken(t *testing.T) {
	cases := []struct {
		tok  string
		want bool
	}{
		{legacyOld, true},
		{"AT_0123456789ABCDEF0123456789ABCDEF", true},
		{"DE_0123456789abcdef0123456789abcde", false},  // 35 chars
		{"DE_0123456789abcdef0123456789abcdeg", false}, // non-hex
		{"de_0123456789abcdef0123456789abcdef", false},
		{fakeJWT(1), false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsLegacyToken(c.tok); got != c.want {
			t.Errorf("IsLegacyToken(%q) = %v, want %v", c.tok, got, c.want)
		}
	}
}

func TestNeedsRefresh_OAuth2Token(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		expiry time.Duration
		want   bool
	}{
		{"expires in 1h", time.Hour, false},
		{"expires in 6m", 6 * time.Minute, false},
		{"expires in 5m exactly", 5 * time.Minute, true},
		{"expires in 1m", time.Minute, true},
	}
	for _, c := range cases {
		tok := &Token{AccessToken: fakeJWT(0), ExpiresAt: now.Add(c.expiry)}
		if got := NeedsRefresh(tok, now); got != c.want {
			t.Errorf("%s: NeedsRefresh = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRecoverFromConfig_JWTExpiry(t *testing.T) {
	exp := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	m := NewManager()
	m.SetNowFunc(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) })
	m.RecoverFromConfig(config.Config{Miele: config.MieleConfig{
		Token: &config.TokenConfig{Access: fakeJWT(exp.Unix()), Refresh: "r"},
	}})
	if got := m.Current().ExpiresAt; !got.Equal(exp) {
		t.Errorf("ExpiresAt = %v, want %v (from JWT exp)", got, exp)
	}
}

func TestAuthorizeURL(t *testing.T) {
	c := NewClient()
	cfg := config.MieleConfig{ClientID: "cid", Scope: "openid mcs_thirdparty_read"}
	raw := c.AuthorizeURL(cfg, "http://pi:6061/callback", "st", "chal")
	if !strings.Contains(raw, "scope=openid%20mcs_thirdparty_read") {
		t.Errorf("scope not %%20-separated: %s", raw)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Scheme + "://" + u.Host + u.Path; got != AuthBaseURL+"/auth" {
		t.Errorf("endpoint = %q", got)
	}
	q := u.Query()
	want := map[string]string{
		"client_id":             "cid",
		"response_type":         "code",
		"redirect_uri":          "http://pi:6061/callback",
		"scope":                 "openid mcs_thirdparty_read",
		"state":                 "st",
		"code_challenge":        "chal",
		"code_challenge_method": "S256",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
}

func TestExchangeCodeAndRefreshOAuth2(t *testing.T) {
	legacy := &tokenRecorder{}
	oauth2 := &tokenRecorder{body: `{"access_token":"a","refresh_token":"r","token_type":"Bearer","expires_in":300,"refresh_expires_in":2592000}`}
	lc, _ := stubBoth(t, legacy, oauth2, nil)
	cfg := config.MieleConfig{ClientID: "cid", ClientSecret: "csec"}

	res, err := lc.ExchangeCode(context.Background(), cfg, "C", "http://pi:6061/callback", "VER")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if res.AccessToken != "a" || res.ExpiresIn != 300 || res.RefreshExpiresIn != 2592000 {
		t.Errorf("token = %+v", res)
	}
	if _, err := lc.RefreshOAuth2(context.Background(), cfg, "RT"); err != nil {
		t.Fatalf("RefreshOAuth2: %v", err)
	}

	calls := oauth2.calls()
	if len(calls) != 2 || len(legacy.calls()) != 0 {
		t.Fatalf("oauth2 calls = %d, legacy calls = %d", len(calls), len(legacy.calls()))
	}
	ex := calls[0]
	for k, v := range map[string]string{"grant_type": "authorization_code", "code": "C", "redirect_uri": "http://pi:6061/callback", "code_verifier": "VER", "client_id": "cid", "client_secret": "csec"} {
		if ex.Get(k) != v {
			t.Errorf("exchange %s = %q, want %q", k, ex.Get(k), v)
		}
	}
	rf := calls[1]
	if rf.Get("grant_type") != "refresh_token" || rf.Get("refresh_token") != "RT" {
		t.Errorf("refresh form = %v", rf)
	}
}

func TestLogin_RefreshRouteByTokenFormat(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		access     string
		wantLegacy bool
	}{
		{"legacy token refreshes at legacy endpoint", legacyOld, true},
		{"jwt token refreshes at oauth2 endpoint", fakeJWT(now.Unix()), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ok := `{"access_token":"new","refresh_token":"r2","token_type":"Bearer","expires_in":3600}`
			legacy := &tokenRecorder{body: ok}
			oauth2 := &tokenRecorder{body: ok}
			lc, apiC := stubBoth(t, legacy, oauth2, nil)
			loadTestConfig(t, `{"client-id": "cid", "client-secret": "csec", "persistToken": false}`)

			m := NewManager()
			m.SetClients(lc, apiC)
			m.SetNowFunc(func() time.Time { return now })
			m.SetAuthorizer(func(context.Context, config.MieleConfig, *Client) (TokenResult, error) {
				t.Fatal("browser login must not be started")
				return TokenResult{}, nil
			})
			m.SetToken(&Token{AccessToken: c.access, RefreshToken: "r", ExpiresAt: now.Add(time.Minute)})

			if _, err := m.Login(context.Background()); err != nil {
				t.Fatalf("Login: %v", err)
			}
			gotLegacy, gotOAuth2 := len(legacy.calls()), len(oauth2.calls())
			if c.wantLegacy && (gotLegacy != 1 || gotOAuth2 != 0) || !c.wantLegacy && (gotLegacy != 0 || gotOAuth2 != 1) {
				t.Errorf("legacy calls = %d, oauth2 calls = %d", gotLegacy, gotOAuth2)
			}
		})
	}
}

func TestLogin_DeviceMode_RefreshFailureStartsBrowserLogin(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	legacy := &tokenRecorder{}
	oauth2 := &tokenRecorder{} // refresh refused
	lc, apiC := stubBoth(t, legacy, oauth2, nil)
	loadTestConfig(t, `{"client-id": "cid", "client-secret": "csec", "persistToken": false}`)

	m := NewManager()
	m.SetClients(lc, apiC)
	m.SetNowFunc(func() time.Time { return now })
	authRequired := 0
	m.SetOnAuthRequired(func() { authRequired++ })
	m.SetAuthorizer(func(context.Context, config.MieleConfig, *Client) (TokenResult, error) {
		return TokenResult{AccessToken: "browser", RefreshToken: "r", ExpiresIn: 300}, nil
	})
	m.SetToken(&Token{AccessToken: fakeJWT(now.Unix()), RefreshToken: "r", ExpiresAt: now})

	tok, err := m.Login(context.Background())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok.AccessToken != "browser" || authRequired != 1 || len(oauth2.calls()) != 1 {
		t.Errorf("token = %q, authRequired = %d, oauth2 calls = %d", tok.AccessToken, authRequired, len(oauth2.calls()))
	}
}

func TestLogin_ExplicitDeviceDiscardsLegacyToken(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	legacy := &tokenRecorder{body: `{"access_token":"x","refresh_token":"r","expires_in":3600}`}
	oauth2 := &tokenRecorder{}
	devicesOK := func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{}`)) }
	lc, apiC := stubBoth(t, legacy, oauth2, devicesOK)
	loadTestConfig(t, `{"client-id": "cid", "client-secret": "csec", "auth-type": "per-device", "username": "u", "password": "p", "persistToken": false}`)

	m := NewManager()
	m.SetClients(lc, apiC)
	m.SetNowFunc(func() time.Time { return now })
	browser := 0
	m.SetAuthorizer(func(context.Context, config.MieleConfig, *Client) (TokenResult, error) {
		browser++
		return TokenResult{AccessToken: "browser", RefreshToken: "r", ExpiresIn: 300}, nil
	})
	m.SetToken(&Token{AccessToken: legacyOld, RefreshToken: "r", ExpiresAt: now.Add(48 * time.Hour)})

	tok, err := m.Login(context.Background())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok.AccessToken != "browser" || browser != 1 || len(legacy.calls()) != 0 {
		t.Errorf("token = %q, browser = %d, legacy calls = %d", tok.AccessToken, browser, len(legacy.calls()))
	}
}

func TestLogin_DerivedDeviceKeepsRefreshingLegacyToken(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	legacy := &tokenRecorder{body: `{"access_token":"` + legacyOld + `","refresh_token":"r2","expires_in":86400000}`}
	oauth2 := &tokenRecorder{}
	lc, apiC := stubBoth(t, legacy, oauth2, nil)
	// No auth-type and no credentials: per-device mode, but a stored legacy token
	// is still refreshed at the legacy endpoint.
	loadTestConfig(t, `{"client-id": "cid", "client-secret": "csec", "persistToken": false}`)

	m := NewManager()
	m.SetClients(lc, apiC)
	m.SetNowFunc(func() time.Time { return now })
	m.SetAuthorizer(func(context.Context, config.MieleConfig, *Client) (TokenResult, error) {
		t.Fatal("browser login must not be started")
		return TokenResult{}, nil
	})
	m.SetToken(&Token{AccessToken: legacyOld, RefreshToken: "r", ExpiresAt: now.Add(time.Hour)})

	if _, err := m.Login(context.Background()); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if len(legacy.calls()) != 1 {
		t.Errorf("legacy calls = %d, want 1", len(legacy.calls()))
	}
}

// --- login page ---

// startAuthPage runs authorize on a random port and returns its base URL and
// a channel delivering the result.
func startAuthPage(t *testing.T, cfg config.MieleConfig, lc *Client) (string, <-chan TokenResult, <-chan error, context.CancelFunc) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	resCh := make(chan TokenResult, 1)
	errCh := make(chan error, 1)
	go func() {
		res, err := authorize(ctx, cfg, lc, ln)
		if err != nil {
			errCh <- err
			return
		}
		resCh <- res
	}()
	return "http://" + ln.Addr().String(), resCh, errCh, cancel
}

var authLinkRe = regexp.MustCompile(`href="([^"]+/oidc/auth\?[^"]+)"`)

// openIndex fetches the login page and returns the query of its auth link.
func openIndex(t *testing.T, base string) url.Values {
	t.Helper()
	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	m := authLinkRe.FindStringSubmatch(string(body))
	if m == nil {
		t.Fatalf("auth link missing in page: %s", body)
	}
	u, err := url.Parse(strings.ReplaceAll(m[1], "&amp;", "&"))
	if err != nil {
		t.Fatal(err)
	}
	return u.Query()
}

func waitResult(t *testing.T, resCh <-chan TokenResult, errCh <-chan error) TokenResult {
	t.Helper()
	select {
	case res := <-resCh:
		return res
	case err := <-errCh:
		t.Fatalf("authorize: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("authorize did not return")
	}
	return TokenResult{}
}

func TestAuthPage_CallbackFlow(t *testing.T) {
	oauth2 := &tokenRecorder{body: `{"access_token":"a","refresh_token":"r","token_type":"Bearer","expires_in":300}`}
	lc, _ := stubBoth(t, &tokenRecorder{}, oauth2, nil)
	cfg := config.MieleConfig{ClientID: "cid", ClientSecret: "csec", Scope: "openid"}
	base, resCh, errCh, _ := startAuthPage(t, cfg, lc)

	q := openIndex(t, base)
	wantRedirect := base + "/callback"
	if q.Get("redirect_uri") != wantRedirect {
		t.Errorf("redirect_uri = %q, want %q (derived from request)", q.Get("redirect_uri"), wantRedirect)
	}

	// Wrong state: error page, server keeps running.
	resp, err := http.Get(base + "/callback?code=C&state=bogus")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("wrong state status = %d, want 400", resp.StatusCode)
	}

	resp, err = http.Get(base + "/callback?code=C&state=" + url.QueryEscape(q.Get("state")))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Login successful") {
		t.Errorf("callback status = %d, body = %s", resp.StatusCode, body)
	}

	if res := waitResult(t, resCh, errCh); res.AccessToken != "a" {
		t.Errorf("token = %+v", res)
	}

	ex := oauth2.calls()[0]
	if ex.Get("redirect_uri") != wantRedirect || ex.Get("code") != "C" {
		t.Errorf("exchange form = %v", ex)
	}
	sum := sha256.Sum256([]byte(ex.Get("code_verifier")))
	if base64.RawURLEncoding.EncodeToString(sum[:]) != q.Get("code_challenge") {
		t.Error("code_verifier does not match code_challenge")
	}

	// The page is closed after a successful login.
	if _, err := http.Get(base + "/"); err == nil {
		t.Error("login page still reachable after successful login")
	}
}

func TestAuthPage_PasteFlow(t *testing.T) {
	cases := []struct {
		name  string
		paste func(state string) string
	}{
		{"full redirect URL", func(state string) string {
			return "http://localhost:6061/callback?state=" + url.QueryEscape(state) + "&session_state=x&code=PASTED"
		}},
		{"raw code", func(string) string { return "  PASTED \n" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			oauth2 := &tokenRecorder{body: `{"access_token":"a","refresh_token":"r","expires_in":300}`}
			lc, _ := stubBoth(t, &tokenRecorder{}, oauth2, nil)
			cfg := config.MieleConfig{ClientID: "cid", ClientSecret: "csec", Scope: "openid"}
			base, resCh, errCh, _ := startAuthPage(t, cfg, lc)

			q := openIndex(t, base)
			resp, err := http.PostForm(base+"/code", url.Values{"code": {c.paste(q.Get("state"))}})
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d", resp.StatusCode)
			}
			waitResult(t, resCh, errCh)
			if got := oauth2.calls()[0].Get("code"); got != "PASTED" {
				t.Errorf("code = %q, want PASTED", got)
			}
		})
	}
}

func TestAuthPage_RedirectURIOverrideAndForwardedHeaders(t *testing.T) {
	lc, _ := stubBoth(t, &tokenRecorder{}, &tokenRecorder{}, nil)

	cfg := config.MieleConfig{ClientID: "cid", Scope: "openid", RedirectURI: "https://gw.example/callback"}
	base, _, _, cancel := startAuthPage(t, cfg, lc)
	if got := openIndex(t, base).Get("redirect_uri"); got != "https://gw.example/callback" {
		t.Errorf("override: redirect_uri = %q", got)
	}
	cancel()

	cfg.RedirectURI = ""
	base, _, _, _ = startAuthPage(t, cfg, lc)
	req, _ := http.NewRequest(http.MethodGet, base+"/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "proxy.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), url.QueryEscape("https://proxy.example/callback")) {
		t.Errorf("forwarded headers not honoured: %s", body)
	}
}

func TestAuthPage_ProviderErrorAndCancel(t *testing.T) {
	lc, _ := stubBoth(t, &tokenRecorder{}, &tokenRecorder{}, nil)
	cfg := config.MieleConfig{ClientID: "cid", Scope: "openid"}
	base, _, errCh, cancel := startAuthPage(t, cfg, lc)

	resp, err := http.Get(base + "/callback?error=access_denied&error_description=user+said+no")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "user said no") {
		t.Errorf("status = %d, body = %s", resp.StatusCode, body)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("err = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("authorize did not return after cancel")
	}
}

func TestParsePastedCode(t *testing.T) {
	cases := []struct{ in, code, state string }{
		{"abc", "abc", ""},
		{"http://localhost:6061/callback?state=s&code=c", "c", "s"},
		{"state=s&code=c", "c", "s"},
		{"  c2  ", "c2", ""},
	}
	for _, c := range cases {
		code, state := parsePastedCode(c.in)
		if code != c.code || state != c.state {
			t.Errorf("parsePastedCode(%q) = %q, %q; want %q, %q", c.in, code, state, c.code, c.state)
		}
	}
}
