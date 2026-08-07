package okapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jkaninda/njia"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Standard-library interop. Middleware and handler registration are covered by
// TestStdMiddleware in middlewares_test.go; these tests pin the behavior that is
// specific to standard handlers — parameter access, catch-all mounts, and how
// such routes surface in the OpenAPI document.

// getStd runs a GET through the router and returns the recorder.
func getStd(o *Okapi, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	o.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

// Okapi's router captures path parameters into the request context rather than
// through http.ServeMux, so wrapHTTPHandler bridges them into the request with
// SetPathValue. Both accessors must work inside a standard handler.
func TestStdHandlerPathParams(t *testing.T) {
	o := New()
	var gotParam, gotPathValue string

	o.HandleStd(http.MethodGet, "/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		gotParam = njia.Param(r, "id")
		gotPathValue = r.PathValue("id")
		w.WriteHeader(http.StatusOK)
	})

	rec := getStd(o, "/users/42")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "42", gotParam, "njia.Param should return the captured value")
	assert.Equal(t, "42", gotPathValue, "r.PathValue should be bridged for standard handlers")
}

// Every way of registering a standard handler goes through wrapHTTPHandler, so
// r.PathValue behaves identically for all of them. Group.HandleStd once had its
// own inline conversion and silently skipped the bridge.
func TestStdHandlerPathValueAcrossRegistrations(t *testing.T) {
	o := New()
	got := map[string]string{}

	record := func(key string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			got[key] = r.PathValue("id")
			w.WriteHeader(http.StatusOK)
		}
	}

	o.HandleStd(http.MethodGet, "/a/{id}", record("HandleStd"))
	o.HandleHTTP(http.MethodGet, "/b/{id}", record("HandleHTTP"))

	g := o.Group("/g")
	g.HandleStd(http.MethodGet, "/c/{id}", record("Group.HandleStd"))
	g.HandleHTTP(http.MethodGet, "/d/{id}", record("Group.HandleHTTP"))

	for _, path := range []string{"/a/1", "/b/2", "/g/c/3", "/g/d/4"} {
		require.Equal(t, http.StatusOK, getStd(o, path).Code, path)
	}

	assert.Equal(t, "1", got["HandleStd"])
	assert.Equal(t, "2", got["HandleHTTP"])
	assert.Equal(t, "3", got["Group.HandleStd"])
	assert.Equal(t, "4", got["Group.HandleHTTP"])
}

// Multiple parameters are all bridged, in pattern order.
func TestStdHandlerMultiplePathValues(t *testing.T) {
	o := New()
	var x, y, z string

	o.HandleStd(http.MethodGet, "/a/{x}/b/{y}/c/{z}", func(w http.ResponseWriter, r *http.Request) {
		x, y, z = r.PathValue("x"), r.PathValue("y"), r.PathValue("z")
		w.WriteHeader(http.StatusOK)
	})

	require.Equal(t, http.StatusOK, getStd(o, "/a/1/b/2/c/3").Code)
	assert.Equal(t, "1", x)
	assert.Equal(t, "2", y)
	assert.Equal(t, "3", z)
}

// A route without parameters skips the bridge entirely; an unknown name still
// reports empty, as it does under http.ServeMux.
func TestStdHandlerNoPathParams(t *testing.T) {
	o := New()
	var unknown string

	o.HandleStd(http.MethodGet, "/health", func(w http.ResponseWriter, r *http.Request) {
		unknown = r.PathValue("missing")
		w.WriteHeader(http.StatusOK)
	})

	require.Equal(t, http.StatusOK, getStd(o, "/health").Code)
	assert.Empty(t, unknown)
}

// A catch-all segment hands the prefix and everything beneath it to one
// http.Handler, which is how a standard file server is mounted.
func TestHandleHTTPCatchAllServesSubtree(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "css"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "css", "app.css"), []byte("body{}"), 0o644))

	o := New()
	o.HandleHTTP(http.MethodGet, "/assets/{any...}",
		http.StripPrefix("/assets/", http.FileServer(http.Dir(dir))))

	rec := getStd(o, "/assets/css/app.css")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "body{}", rec.Body.String())

	// A miss inside the subtree is the file server's 404, meaning the route matched.
	rec = getStd(o, "/assets/missing.css")
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// The catch-all must not swallow a sibling path that merely shares the prefix.
	assert.NotEqual(t, http.StatusOK, getStd(o, "/assetsxyz").Code)
}

// Standard handlers are ordinary routes, so they reach the OpenAPI document and
// accept the same documentation options. Okapi cannot infer body schemas for
// them, because a standard handler has no typed input or output.
func TestStdHandlerAppearsInOpenAPISpec(t *testing.T) {
	o := New()
	o.HandleStd(http.MethodGet, "/std-documented", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, DocSummary("A standard handler"), DocTag("std"))

	o.buildOpenAPISpec()
	spec := o.openapiSpec
	require.NotNil(t, spec)

	item := spec.Paths.Find("/std-documented")
	require.NotNil(t, item, "standard handler should be documented")
	require.NotNil(t, item.Get)
	assert.Equal(t, "A standard handler", item.Get.Summary)
	assert.Contains(t, item.Get.Tags, "std")
	assert.Nil(t, item.Get.RequestBody, "no typed input, so no request schema")
}

// DocHide keeps a standard handler out of the specification without affecting
// routing.
func TestStdHandlerDocHide(t *testing.T) {
	o := New()
	o.HandleStd(http.MethodGet, "/hidden", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, DocHide())
	// Control route, so an empty spec cannot make this test pass by accident.
	o.HandleStd(http.MethodGet, "/visible", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	o.buildOpenAPISpec()
	require.NotNil(t, o.openapiSpec)
	require.NotNil(t, o.openapiSpec.Paths.Find("/visible"), "control route should be documented")
	assert.Nil(t, o.openapiSpec.Paths.Find("/hidden"), "DocHide should omit the route")

	// Hidden from the docs, but still routable.
	assert.Equal(t, http.StatusOK, getStd(o, "/hidden").Code)
}
