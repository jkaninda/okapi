package okapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jkaninda/okapi/okapitest"
)

func TestCORSHandler_AddsVaryHeaders(t *testing.T) {
	cors := Cors{
		AllowedOrigins: []string{"https://app.example"},
	}

	ctx, rec := NewTestContext(http.MethodOptions, "http://example.test/books", nil)
	ctx.okapi = New(WithAccessLogDisabled())
	ctx.request.Header.Set("Origin", "https://app.example")
	ctx.request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	ctx.request.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

	called := false
	err := cors.CORSHandler(func(c *Context) error {
		called = true
		return nil
	})(ctx)
	if err != nil {
		t.Fatalf("CORS handler returned error: %v", err)
	}
	if called {
		t.Fatal("expected preflight request to be handled without calling next")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	vary := rec.Header().Get("Vary")
	for _, expected := range []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"} {
		if !strings.Contains(vary, expected) {
			t.Fatalf("expected Vary header to contain %q, got %q", expected, vary)
		}
	}
}

func TestWithCORS_PreflightAddsVaryHeaders(t *testing.T) {
	app := New(
		WithAccessLogDisabled(),
		WithCors(Cors{
			AllowedOrigins: []string{"*"},
			AllowMethods:   []string{http.MethodGet, http.MethodPost},
		}),
	)
	app.Get("/books", func(c *Context) error {
		return c.OK(M{"ok": true})
	})

	baseURL := app.StartForTest(t)

	okapitest.OPTIONS(t, baseURL+"/books").
		Header("Origin", "https://app.example").
		Header("Access-Control-Request-Method", http.MethodGet).
		Header("Access-Control-Request-Headers", "Authorization").
		ExpectStatusNoContent().
		ExpectHeaderContains("Vary", "Origin").
		ExpectHeaderContains("Vary", "Access-Control-Request-Method").
		ExpectHeaderContains("Vary", "Access-Control-Request-Headers")
}
