/*
 *  MIT License
 *
 * Copyright (c) 2025 Jonas Kaninda
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in all
 *  copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 *  SOFTWARE.
 */

package okapi

import (
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

// noDirListing wraps http.FileSystem to disable directory listing
type noDirListing struct {
	fs http.FileSystem
}

func (n noDirListing) Open(name string) (http.File, error) {
	f, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// If it's a directory and has no index.html, block it
	if stat.IsDir() {
		_, err := n.fs.Open(path.Join(name, "index.html"))
		if err != nil {
			return nil, fs.ErrNotExist
		}
	}
	return f, nil
}

// WebConfig configures how a web application (typically a single-page
// application) is served by Okapi.Web and Okapi.WebFS. The zero value is valid
// and produces sensible defaults (serve index.html, auto-exclude registered
// API routes).
type WebConfig struct {
	// Index is the file served for client-side routes that do not map to a
	// real file on disk. Defaults to "index.html".
	Index string

	// Root is the sub-directory inside the http.FileSystem / fs.FS that holds
	// the built application. It is only used by WebFS (e.g. "web/dist" for an
	// embed.FS rooted at the module). Ignored by Web.
	Root string

	// Exclude lists additional path prefixes that must never fall back to the
	// index document.
	Exclude []string

	// DisableAutoExclude turns off the default behaviour of excluding the
	// top-level segment of every registered route from the index fallback. When
	// true, only Exclude is consulted.
	DisableAutoExclude bool

	// MaxAge sets the Cache-Control max-age for real asset files (JS, CSS,
	// images...). The index document is always served with "no-cache" so a
	// new deploy is picked up immediately. Zero means no Cache-Control header
	// is added for assets.
	MaxAge time.Duration
}

// SPAConfig configures how a single-page application is served.
//
// Deprecated: use WebConfig instead.
type SPAConfig = WebConfig

func resolveWebConfig(cfg ...WebConfig) WebConfig {
	c := WebConfig{}
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.Index == "" {
		c.Index = constIndex
	}
	return c
}

// Web serves a web application (typically a single-page application) from a
// directory on disk.
//
// Real files (assets, the index, etc.) are served directly; any unmatched
// path that is not excluded falls back to the index document so the
// client-side router can take over. Registered API routes keep precedence,
// and their top-level path segments are excluded from the fallback
// automatically (see WebConfig.DisableAutoExclude).
//
// Web should be registered after your API routes so they win the match.
//
// Example:
//
//	app.Get("/api/v1/users", listUsers)
//	app.Web("/", "./web") // serves ./web/index.html for "/", "/login", ...
func (o *Okapi) Web(prefix, dir string, cfg ...WebConfig) {
	o.webHandler(prefix, http.Dir(dir), resolveWebConfig(cfg...))
}

// WebFS serves a web application (typically a single-page application) from an
// fs.FS, such as an embed.FS.
//
// Example:
//
//	//go:embed all:web/dist
//	var dist embed.FS
//
//	app.Get("/api/v1/users", listUsers)
//	app.WebFS("/", dist, okapi.WebConfig{Root: "web/dist"})
func (o *Okapi) WebFS(prefix string, fsys fs.FS, cfg ...WebConfig) {
	c := resolveWebConfig(cfg...)
	if c.Root != "" {
		sub, err := fs.Sub(fsys, c.Root)
		if err != nil {
			fPrintError("WebFS: invalid Root, serving filesystem root instead ", "root", c.Root, "error", err)
		} else {
			fsys = sub
		}
	}
	if _, err := fs.Stat(fsys, strings.TrimPrefix(c.Index, "/")); err != nil {
		fPrintError("WebFS: index file not found, the app will fall back to 404 ", "root", c.Root, "index", c.Index, "error", err)
	}
	o.webHandler(prefix, http.FS(fsys), c)
}

// SPA serves a single-page application from a directory on disk.
//
// Deprecated: use Web instead.
func (o *Okapi) SPA(prefix, dir string, cfg ...WebConfig) {
	o.Web(prefix, dir, cfg...)
}

// SPAFS serves a single-page application from an fs.FS.
//
// Deprecated: use WebFS instead.
func (o *Okapi) SPAFS(prefix string, fsys fs.FS, cfg ...WebConfig) {
	o.WebFS(prefix, fsys, cfg...)
}

// webHandler registers the index-fallback handler under prefix.
func (o *Okapi) webHandler(prefix string, root http.FileSystem, c WebConfig) {
	if prefix == "" {
		prefix = "/"
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		if o.webExcluded(r.URL.Path, prefix, c) {
			o.webNotFound(w, r)
			return
		}

		rel := strings.TrimPrefix(r.URL.Path, strings.TrimSuffix(prefix, "/"))
		clean := path.Clean("/" + strings.TrimPrefix(rel, "/"))

		if clean != "/" && serveWebFile(w, r, root, clean, c.MaxAge) {
			return
		}
		serveWebIndex(w, r, root, c.Index)
	}
	o.router.muxRouter.PathPrefix(prefix).
		HandlerFunc(handler).
		Methods(http.MethodGet, http.MethodHead)
}

func (o *Okapi) webExcluded(urlPath, prefix string, c WebConfig) bool {
	for _, ex := range c.Exclude {
		ex = strings.TrimSuffix(ex, "/")
		if ex != "" && (urlPath == ex || strings.HasPrefix(urlPath, ex+"/")) {
			return true
		}
	}
	if c.DisableAutoExclude {
		return false
	}
	seg := firstPathSegment(urlPath)
	if seg == "" || seg == firstPathSegment(prefix) {
		return false
	}
	for _, rt := range o.routes {
		if firstPathSegment(rt.Path) == seg {
			return true
		}
	}
	return false
}

func (o *Okapi) webNotFound(w http.ResponseWriter, r *http.Request) {
	if h := o.router.muxRouter.NotFoundHandler; h != nil {
		h.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

func serveWebFile(w http.ResponseWriter, r *http.Request, root http.FileSystem, name string, maxAge time.Duration) bool {
	f, err := root.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		return false
	}
	if maxAge > 0 {
		w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(maxAge.Seconds())))
	}
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
	return true
}

func serveWebIndex(w http.ResponseWriter, r *http.Request, root http.FileSystem, index string) {
	f, err := root.Open("/" + strings.TrimPrefix(index, "/"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
}

func firstPathSegment(p string) string {
	p = strings.TrimPrefix(p, "/")
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return p
}
