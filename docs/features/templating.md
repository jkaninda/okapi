---
title: Templating
layout: default
parent: Features
nav_order: 4
---

# Templating

Okapi supports template rendering for serving HTML pages. You can use the built-in renderer, Go's standard `html/template` package, or any custom renderer that implements Okapi's `Renderer` interface.


## Built-in Template Renderer

The simplest way to get started. `NewTemplateFromDirectory` scans a folder for files matching the given extensions and registers them by filename (without extension) as named templates.
```go
func main() {
    // Load all .html and .tmpl files from the views directory
    tmpl, err := okapi.NewTemplateFromDirectory("public/views", ".html", ".tmpl")
    if err != nil {
        log.Fatal(err)
    }

    o := okapi.Default().WithRenderer(tmpl)

    o.Get("/", func(c *okapi.Context) error {
        return c.Render(http.StatusOK, "hello", okapi.M{
            "title":   "Greeting Page",
            "message": "Hello, World!",
        })
    })

    if err := o.Start(); err != nil {
        log.Fatal(err)
    }
}
```


You can also load templates from a glob pattern instead of a directory:
```go
tmpl, err := okapi.NewTemplateFromFiles("public/views/*.html")
```

---

## Embedded Templates

For production deployments, embedding templates directly into the binary using Go's `embed` package eliminates runtime file-system dependencies.
```go
var (
    //go:embed views/*
    Views embed.FS

    AssetsFS = http.FS(must(fs.Sub(Views, "views/assets")))
)

func main() {
    app := okapi.New()
    app.WithRendererFromFS(Views, "views/*.html")

    app.Get("/", func(c *okapi.Context) error {
        return c.Render(http.StatusOK, "home", okapi.M{
            "title":    "Go Okapi Bookstore",
            "headline": "Discover your next great read",
            "books":    books,
        })
    })

    // Serve embedded static assets
    app.StaticFS("/assets", AssetsFS)

    if err := app.Start(); err != nil {
        panic(err)
    }
}
func must(fsys fs.FS, err error) fs.FS {
    if err != nil {
    panic(err)
    }
    return fsys
}

```

---

## Custom Renderers

If the built-in renderer doesn't fit your needs — for example, you want to use a third-party templating engine like Sprig or Petal — you can supply your own. Okapi supports two approaches.

### Renderer Function

Use `RendererFunc` for lightweight, one-off renderers without needing a full struct:
```go
o.Renderer = okapi.RendererFunc(func(w io.Writer, name string, data interface{}, c *okapi.Context) error {
    tmpl, err := template.ParseFiles("templates/" + name + ".html")
    if err != nil {
        return err
    }
    return tmpl.ExecuteTemplate(w, name, data)
})
```

> **Warning:** This example re-parses the template file on every request. In production, parse templates once at startup and cache the result (see the struct-based example below).

### Struct-Based Renderer

For more control — caching, shared state, or preloaded templates — implement the `Renderer` interface on a struct:
```go
type Template struct {
    templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c *okapi.Context) error {
    return t.templates.ExecuteTemplate(w, name, data)
}

// At startup:
tmpl := &Template{
    templates: template.Must(template.ParseGlob("templates/*.html")),
}

o := okapi.New().WithRenderer(tmpl)
```

Templates are parsed once via `ParseGlob` and reused for every request, which is the recommended approach for production.

---

## Rendering a View

Once a renderer is attached, use `c.Render` inside any handler to render a named template with arbitrary data:
```go
o.Get("/welcome", func(c *okapi.Context) error {
    return c.Render(http.StatusOK, "welcome", okapi.M{
        "title":   "Welcome Page",
        "message": "Hello from Okapi!",
    })
})
```

`okapi.M` is a shorthand for `map[string]interface{}`. Each key becomes accessible inside the template as `{{.keyName}}`.

### Example Template

`templates/welcome.html`:
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>{{.title}}</title>
</head>
<body>
    <h1>{{.message}}</h1>
</body>
</html>
```

---

## Static File Serving

Okapi can serve static assets alongside your rendered pages.

### Serve a directory

All files under `public/assets` become accessible at `/static/*`:
```go
o.Static("/static", "public/assets")
```

### Serve a single file
```go
o.Get("/favicon.ico", func(c *okapi.Context) error {
    c.ServeFile("public/favicon.ico")
    return nil
})
```

### Serve an embedded filesystem

Use `StaticFS` to serve assets that were compiled into the binary via `embed`:
```go
app.StaticFS("/assets", AssetsFS)
```

This pairs naturally with the [Embedded Templates](#embedded-templates) example above, where `AssetsFS` is derived from the same `embed.FS`.