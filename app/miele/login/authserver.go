package login

import (
	"context"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
	"github.com/philipparndt/go-logger"
)

// maxPendingLogins bounds the number of remembered login attempts (one per
// view of the login page).
const maxPendingLogins = 10

// pendingLogin is one started browser login, keyed by its OAuth2 state.
type pendingLogin struct {
	state       string
	redirectURI string
	pkce        pkce
}

// authServer is the temporary login page served while a browser login is
// pending. It renders plain HTML so it works in text browsers (w3m, lynx).
type authServer struct {
	cfg    config.MieleConfig
	client *Client

	mu      sync.Mutex
	pending []pendingLogin

	once   sync.Once
	result chan TokenResult
}

// Authorize runs the browser login of the OAuth2 provider: it serves a login
// page on cfg.AuthListen until the user has logged in and the authorization
// code has been exchanged, then shuts the page down and returns the tokens.
func Authorize(ctx context.Context, cfg config.MieleConfig, c *Client) (TokenResult, error) {
	ln, err := net.Listen("tcp", cfg.AuthListen)
	if err != nil {
		return TokenResult{}, fmt.Errorf("login page: %w", err)
	}
	return authorize(ctx, cfg, c, ln)
}

func authorize(ctx context.Context, cfg config.MieleConfig, c *Client, ln net.Listener) (TokenResult, error) {
	s := &authServer{cfg: cfg, client: c, result: make(chan TokenResult, 1)}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /callback", s.handleCallback)
	mux.HandleFunc("POST /code", s.handleCode)
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	logger.Info("Miele authorization required. To login, visit now: " + strings.Join(loginPageURLs(ln.Addr()), " or "))
	if cfg.RedirectURI != "" {
		if p, err := s.newPending(cfg.RedirectURI); err == nil {
			logger.Info("Or open this URL in any browser: " + c.AuthorizeURL(cfg, p.redirectURI, p.state, p.pkce.Challenge))
		}
	}

	shutdown := func() {
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}

	select {
	case res := <-s.result:
		shutdown()
		logger.Info("Browser login completed. Login page closed.")
		return res, nil
	case <-ctx.Done():
		shutdown()
		return TokenResult{}, ctx.Err()
	case err := <-serveErr:
		return TokenResult{}, fmt.Errorf("login page: %w", err)
	}
}

// loginPageURLs lists URLs under which the login page is likely reachable.
func loginPageURLs(addr net.Addr) []string {
	tcp, ok := addr.(*net.TCPAddr)
	if !ok {
		return []string{"http://" + addr.String() + "/"}
	}
	port := fmt.Sprint(tcp.Port)
	if !tcp.IP.IsUnspecified() {
		return []string{"http://" + net.JoinHostPort(tcp.IP.String(), port) + "/"}
	}
	urls := []string{"http://localhost:" + port + "/"}
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			urls = append(urls, "http://"+net.JoinHostPort(ipnet.IP.String(), port)+"/")
		}
	}
	return urls
}

// redirectURIFor returns the configured redirect URI, or derives it from the
// request so the browser returns to the same host it used for the login page.
func (s *authServer) redirectURIFor(r *http.Request) string {
	if s.cfg.RedirectURI != "" {
		return s.cfg.RedirectURI
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		scheme = p
	}
	host := r.Host
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}
	return scheme + "://" + host + "/callback"
}

func (s *authServer) newPending(redirectURI string) (pendingLogin, error) {
	state, err := randomString(16)
	if err != nil {
		return pendingLogin{}, err
	}
	pk, err := newPKCE()
	if err != nil {
		return pendingLogin{}, err
	}
	p := pendingLogin{state: state, redirectURI: redirectURI, pkce: pk}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = append(s.pending, p)
	if len(s.pending) > maxPendingLogins {
		s.pending = s.pending[len(s.pending)-maxPendingLogins:]
	}
	return p, nil
}

// lookup finds the pending login for state. An empty state (a pasted raw
// code) selects the most recent login.
func (s *authServer) lookup(state string) (pendingLogin, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.pending) - 1; i >= 0; i-- {
		if state == "" || s.pending[i].state == state {
			return s.pending[i], true
		}
	}
	return pendingLogin{}, false
}

func (s *authServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	p, err := s.newPending(s.redirectURIFor(r))
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, err.Error())
		return
	}
	render(w, http.StatusOK, indexPage, map[string]string{
		"AuthURL":     s.client.AuthorizeURL(s.cfg, p.redirectURI, p.state, p.pkce.Challenge),
		"RedirectURI": p.redirectURI,
	})
}

func (s *authServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		s.renderError(w, http.StatusBadRequest, strings.TrimSpace(e+": "+q.Get("error_description")))
		return
	}
	s.exchange(w, r, q.Get("code"), q.Get("state"))
}

func (s *authServer) handleCode(w http.ResponseWriter, r *http.Request) {
	code, state := parsePastedCode(r.FormValue("code"))
	s.exchange(w, r, code, state)
}

// parsePastedCode accepts the URL the browser was redirected to (or just its
// query string) as well as a bare authorization code.
func parsePastedCode(in string) (code, state string) {
	in = strings.TrimSpace(in)
	if !strings.Contains(in, "code=") {
		return in, ""
	}
	query := in
	if i := strings.Index(in, "?"); i >= 0 {
		query = in[i+1:]
	}
	q, err := url.ParseQuery(query)
	if err != nil {
		return "", ""
	}
	return q.Get("code"), q.Get("state")
}

func (s *authServer) exchange(w http.ResponseWriter, r *http.Request, code, state string) {
	if code == "" {
		s.renderError(w, http.StatusBadRequest, "No authorization code received.")
		return
	}
	p, ok := s.lookup(state)
	if !ok {
		s.renderError(w, http.StatusBadRequest, "Unknown or expired login attempt. Please start the login again.")
		return
	}
	res, err := s.client.ExchangeCode(r.Context(), s.cfg, code, p.redirectURI, p.pkce.Verifier)
	if err != nil {
		logger.Error("Exchanging the authorization code failed", "error", err)
		s.renderError(w, http.StatusBadGateway, "Exchanging the authorization code failed: "+err.Error())
		return
	}
	render(w, http.StatusOK, successPage, nil)
	s.once.Do(func() { s.result <- res })
}

func (s *authServer) renderError(w http.ResponseWriter, status int, msg string) {
	render(w, status, errorPage, map[string]string{"Message": msg})
}

func render(w http.ResponseWriter, status int, t *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.Execute(w, data); err != nil {
		logger.Error("Rendering login page failed", "error", err)
	}
}

var (
	indexPage = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Miele login</title></head>
<body>
<h1>Miele login</h1>
<p>miele-to-mqtt-gw needs access to your Miele appliances.</p>
<p><a href="{{.AuthURL}}">Login with Miele</a></p>
<p>After the login you are redirected back to this page.</p>
<hr>
<p>If the redirect does not reach this page, copy the URL you were redirected to
(it starts with {{.RedirectURI}}) or just its code, and submit it here:</p>
<form method="post" action="/code">
<input type="text" name="code" size="60">
<input type="submit" value="Submit">
</form>
</body></html>
`))

	successPage = template.Must(template.New("success").Parse(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Miele login</title></head>
<body>
<h1>Login successful</h1>
<p>The gateway is now connected to Miele. This page will now close.</p>
</body></html>
`))

	errorPage = template.Must(template.New("error").Parse(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Miele login</title></head>
<body>
<h1>Login failed</h1>
<p>{{.Message}}</p>
<p><a href="/">Try again</a></p>
</body></html>
`))
)
