---
title: Single-Page Applications
layout: default
parent: Features
nav_order: 7
---

# Single-Page Applications (SPA)

Okapi can serve a single-page application (React, Vue, Svelte, Angular, …)
alongside your API. Real files (the index document, JS, CSS, images) are
served directly, and any unmatched path falls back to the SPA index so the
client-side router can take over.

Two entry points are provided:

- `Web(prefix, dir)` — serve from a directory on disk.
- `WebFS(prefix, fsys)` — serve from any `fs.FS`, including an `embed.FS`
  for single-binary deployments.

Register the handler **after** your API routes so the API keeps precedence.

> The `SPA` / `SPAFS` / `SPAConfig` names are deprecated aliases of
> `Web` / `WebFS` / `WebConfig` and continue to work unchanged.

## Serve from disk

Useful during front-end development:

```go
func main() {
    o := okapi.Default()

    o.Get("/api/v1/users", listUsers)

    // Serves ./web/index.html for "/", "/login", "/users/42", ...
    o.Web("/", "./web")

    o.Start()
}
```

## Serve from an embedded filesystem

The recommended approach for production — the whole front-end ships inside
the binary:

```go
//go:embed all:web/dist
var dist embed.FS

func main() {
    o := okapi.Default()

    o.Get("/api/v1/users", listUsers)

    o.WebFS("/", dist, okapi.WebConfig{
        Root:   "web/dist", // sub-directory inside the embed.FS
        MaxAge: time.Hour,  // Cache-Control for assets
    })

    o.Start()
}
```

## How routing works

For every unmatched `GET`/`HEAD` request under the prefix:

1. If the path maps to a real file, that file is served.
2. Otherwise the SPA index document is returned so the client-side router
   can handle the route.

Registered API routes always win the match. In addition, the top-level path
segment of every registered route is **auto-excluded** from the fallback, so
an unknown path under an API namespace returns `404` instead of silently
serving the index. For example, with `/api/v1/users` registered,
`/api/v1/missing` returns `404` rather than `index.html`.

## Caching

- The **index document** is always served with `Cache-Control: no-cache`, so
  a new deploy is picked up immediately.
- **Asset files** honour `WebConfig.MaxAge`. With `MaxAge: time.Hour`, assets
  are served with `Cache-Control: public, max-age=3600`. When `MaxAge` is
  zero, no `Cache-Control` header is added for assets.

## Configuration

`WebConfig` is optional — the zero value serves `index.html` and
auto-excludes registered API routes.

| Field | Description |
| --- | --- |
| `Index` | File served for client-side routes. Defaults to `index.html`. |
| `Root` | Sub-directory inside the `fs.FS` that holds the built app (`WebFS` only). |
| `Exclude` | Additional path prefixes that must never fall back to the index. |
| `DisableAutoExclude` | Turn off auto-excluding registered route segments; only `Exclude` is consulted. |
| `MaxAge` | `Cache-Control` max-age for asset files. The index is always `no-cache`. |

### Excluding extra paths

```go
o.Web("/", "./web", okapi.WebConfig{
    Exclude: []string{"/metrics", "/healthz"},
})
```

A full, runnable example lives in
[`examples/web`](https://github.com/jkaninda/okapi/tree/main/examples/web).
