package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchDevices_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != DevicesPath {
			t.Errorf("path = %q, want %q", r.URL.Path, DevicesPath)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q, want Bearer tok", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"dev-a":{"x":1},"dev-b":{"y":"foo"}}`))
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), BaseURL: srv.URL}
	devs, err := c.FetchDevices(context.Background(), "tok")
	if err != nil {
		t.Fatalf("FetchDevices: %v", err)
	}
	if len(devs) != 2 {
		t.Fatalf("len = %d, want 2", len(devs))
	}
	byID := map[string]string{}
	for _, d := range devs {
		byID[d.ID] = string(d.Data)
	}
	if byID["dev-a"] != `{"x":1}` {
		t.Errorf("dev-a data = %q", byID["dev-a"])
	}
	if byID["dev-b"] != `{"y":"foo"}` {
		t.Errorf("dev-b data = %q", byID["dev-b"])
	}
}

func TestFetchDevices_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), BaseURL: srv.URL}
	_, err := c.FetchDevices(context.Background(), "tok")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error = %v, want to contain status 500", err)
	}
}

func TestFetchDevices_EmptyToken(t *testing.T) {
	c := NewClient()
	_, err := c.FetchDevices(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestPing(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ok.Close)
	c := &Client{HTTP: ok.Client(), BaseURL: ok.URL}
	if !c.Ping(context.Background()) {
		t.Error("Ping = false, want true")
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	t.Cleanup(bad.Close)
	c2 := &Client{HTTP: bad.Client(), BaseURL: bad.URL}
	if c2.Ping(context.Background()) {
		t.Error("Ping = true, want false for 502")
	}
}
