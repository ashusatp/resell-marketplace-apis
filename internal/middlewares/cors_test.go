package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	cors := CORSMiddleware([]string{"http://localhost:5173/", "http://example.com"})(dummyHandler)

	// 1. Allowed origin preflight
	reqPreflight := httptest.NewRequest(http.MethodOptions, "/listings", nil)
	reqPreflight.Header.Set("Origin", "http://localhost:5173")
	recPreflight := httptest.NewRecorder()

	cors.ServeHTTP(recPreflight, reqPreflight)

	if recPreflight.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", recPreflight.Code)
	}
	if recPreflight.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("expected Access-Control-Allow-Origin header, got %q", recPreflight.Header().Get("Access-Control-Allow-Origin"))
	}

	// 2. Disallowed origin preflight
	reqBadPreflight := httptest.NewRequest(http.MethodOptions, "/listings", nil)
	reqBadPreflight.Header.Set("Origin", "http://malicious.com")
	recBadPreflight := httptest.NewRecorder()

	cors.ServeHTTP(recBadPreflight, reqBadPreflight)
	if recBadPreflight.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for bad origin preflight, got %d", recBadPreflight.Code)
	}

	// 3. Allowed origin normal request
	reqGet := httptest.NewRequest(http.MethodGet, "/listings", nil)
	reqGet.Header.Set("Origin", "http://localhost:5173")
	recGet := httptest.NewRecorder()

	cors.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 for allowed GET, got %d", recGet.Code)
	}
	if recGet.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("expected Access-Control-Allow-Origin header, got %q", recGet.Header().Get("Access-Control-Allow-Origin"))
	}
}
