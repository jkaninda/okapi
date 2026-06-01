package okapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jkaninda/okapi/okapitest"
)

func writeSPAFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	files := map[string]string{
		"index.html":    "<!doctype html><title>app</title>",
		"assets/app.js": "console.log('hi')",
		"favicon.ico":   "icon",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func serveSPARequest(o *Okapi, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	o.ServeHTTP(rec, req)
	return rec
}

func TestSPAServesRealFile(t *testing.T) {
	dir := writeSPAFixture(t)
	o := New()
	o.SPA("/", dir)

	rec := serveSPARequest(o, "/assets/app.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("asset status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "console.log('hi')" {
		t.Fatalf("asset body = %q", rec.Body.String())
	}
}

func TestSPAFallsBackToIndex(t *testing.T) {
	dir := writeSPAFixture(t)
	o := New()
	o.SPA("/", dir)

	for _, p := range []string{"/", "/login", "/users/42/profile"} {
		rec := serveSPARequest(o, p)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", p, rec.Code)
		}
		if rec.Body.String() != "<!doctype html><title>app</title>" {
			t.Fatalf("%s did not return the SPA index: %q", p, rec.Body.String())
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
			t.Fatalf("%s Cache-Control = %q, want no-cache", p, cc)
		}
	}
}

func TestSPAAutoExcludesRegisteredRoutes(t *testing.T) {
	dir := writeSPAFixture(t)
	o := New()
	o.Get("/api/v1/users", func(c *Context) error { return c.OK(M{"ok": true}) })
	o.SPA("/", dir)

	if rec := serveSPARequest(o, "/api/v1/users"); rec.Code != http.StatusOK {
		t.Fatalf("registered route status = %d, want 200", rec.Code)
	}
	// An unmatched path under the same namespace 404s instead of serving index.
	rec := serveSPARequest(o, "/api/v1/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unmatched api path status = %d, want 404", rec.Code)
	}
}

func TestSPAExcludeOption(t *testing.T) {
	dir := writeSPAFixture(t)
	o := New()
	o.SPA("/", dir, SPAConfig{Exclude: []string{"/metrics"}})

	rec := serveSPARequest(o, "/metrics")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("excluded path status = %d, want 404", rec.Code)
	}
	// A normal SPA route is unaffected.
	if rec := serveSPARequest(o, "/dashboard"); rec.Code != http.StatusOK {
		t.Fatalf("spa route status = %d, want 200", rec.Code)
	}
}

func TestSPAFSFromFilesystem(t *testing.T) {
	dir := writeSPAFixture(t)
	o := New()
	o.SPAFS("/", os.DirFS(dir))

	if rec := serveSPARequest(o, "/assets/app.js"); rec.Code != http.StatusOK {
		t.Fatalf("fs asset status = %d, want 200", rec.Code)
	}
	rec := serveSPARequest(o, "/some/client/route")
	if rec.Code != http.StatusOK || rec.Body.String() != "<!doctype html><title>app</title>" {
		t.Fatalf("fs fallback failed: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestSPACustomIndexAndCache(t *testing.T) {
	dir := writeSPAFixture(t)
	o := New()
	o.SPA("/", dir, SPAConfig{MaxAge: time.Hour})

	// Index is no-cache.
	if cc := serveSPARequest(o, "/route").Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("index Cache-Control = %q, want no-cache", cc)
	}
	// Real assets get the configured max-age.
	if cc := serveSPARequest(o, "/assets/app.js").Header().Get("Cache-Control"); cc != "public, max-age=3600" {
		t.Fatalf("asset Cache-Control = %q, want public, max-age=3600 (1h)", cc)
	}
}

func TestFirstPathSegment(t *testing.T) {
	cases := map[string]string{
		"/":             "",
		"/api":          "api",
		"/api/v1/users": "api",
		"api/v1":        "api",
		"/{any:.*}":     "{any:.*}",
	}
	for in, want := range cases {
		if got := firstPathSegment(in); got != want {
			t.Errorf("firstPathSegment(%q) = %q, want %q", in, got, want)
		}
	}
}
func TestSPA(t *testing.T) {
	dir := writeSPAFixture(t)
	ts := DefaultTestServer(t)

	ts.Group("/api/v1")
	ts.Get("/health", func(c *Context) error {
		return c.OK(M{"status": "ok"})
	})
	ts.SPA("/", dir, SPAConfig{MaxAge: time.Hour})

	okapitest.GET(t, ts.BaseURL+"/docs").
		ExpectStatusOK()

	okapitest.GET(t, ts.BaseURL+"/swagger").ExpectStatusOK().ExpectBodyContains("swagger-ui")
	okapitest.GET(t, ts.BaseURL+"/redoc").ExpectStatusOK().ExpectBodyContains("redoc")
	okapitest.GET(t, ts.BaseURL+"/scalar").ExpectStatusOK().ExpectBodyContains("@scalar/api-reference")

}
