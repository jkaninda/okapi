---
title: CLI
layout: default
parent: Features
nav_order: 10
---

# CLI Integration

Okapi comes with a built-in command-line interface (CLI) helper that makes it easy to configure and run your application from the terminal.

It supports:

- Typed flags (`string`, `int`, `bool`, `float`,`time.Duration`, etc.)
- Short and long options
- Environment variable binding
- Default values
- Struct-based flag generation
- Graceful lifecycle hooks (start / started / shutdown)

---

## Basic Usage

```go
app := okapi.New()

// Create CLI instance
cli := okapicli.New(app, "myapp").
	String("config", "c", "config.yaml", "Path to configuration file").
	Int("port", "p", 8080, "HTTP server port").
	Bool("debug", "d", false, "Enable debug mode")

// Parse flags
if err := cli.Parse(); err != nil {
	panic(err)
}

// Retrieve flag values
port := cli.GetInt("port")
debug := cli.GetBool("debug")

app.WithPort(port)
app.WithDebug(debug)

app.Get("/", func(ctx *okapi.Context) error {
	return ctx.OK(okapi.M{
		"status":  "ok",
		"message": "CLI example",
	})
})

// Run server with lifecycle hooks
if err := cli.RunServer(&okapicli.RunOptions{
	OnStart: func() {
		slog.Info("Preparing resources before startup")
	},
	OnStarted: func() {
		slog.Info("Server started successfully")
	},
	OnShutdown: func() {
		slog.Info("Cleaning up before shutdown")
	},
}); err != nil {
	panic(err)
}
````

### Example CLI Output

```bash
$ ./myapp --help

Usage:
  myapp [flags]

Flags:
  -c, --config string   Path to configuration file (default "config.yaml")
  -p, --port int        HTTP server port (default 8080)
  -d, --debug           Enable debug mode
```

---

## Struct Tag Support (Automatic Flag Generation)

Instead of manually defining each flag, you can generate flags directly from a struct using tags.

### Supported Tags

| Tag       | Description                            |
|-----------|----------------------------------------|
| `cli`     | Flag name (required)                   |
| `short`   | Short flag name (optional)             |
| `desc`    | Description shown in `--help`          |
| `default` | Default value (string, parsed to type) |
| `env`     | Environment variable name              |

### Example

```go
type ServerConfig struct {
	Port   int    `cli:"port"   short:"p" desc:"HTTP server port"   env:"APP_PORT"   default:"8080"`
	Host   string `cli:"host"   short:"h" desc:"Server hostname"   env:"APP_HOST"   default:"localhost"`
	Debug  bool   `cli:"debug"  short:"d" desc:"Enable debug mode" env:"APP_DEBUG"`
	Config string `cli:"config" short:"c" desc:"Path to config file" default:"config.yaml"`
}

func main() {
	cfg := &ServerConfig{
		Port: 8000, // Struct defaults (lowest priority)
	}

	o := okapi.New()
	cli := okapicli.New(o, "MyApp").
		FromStruct(cfg) // or WithConfig(cfg)

	// Parse: env first, then CLI args (CLI overrides env)
	if err := cli.Parse(); err != nil {
		panic(err)
	}

	slog.Info("Starting server",
		"port", cfg.Port,
		"host", cfg.Host,
		"debug", cfg.Debug,
		"config", cfg.Config,
	)

	// Optional: access flags directly
	port := cli.GetInt("port") // Same as cfg.Port
	_ = port
}
```

---

## Value Resolution Order

When using `FromStruct`, values are resolved in the following order (lowest → highest priority):

1. Struct field default
2. `default` tag
3. Environment variable (`env`)
4. CLI flag

Example:

```text
Port:
  Struct default   → 8000
  APP_PORT=9000    → overrides struct
  --port=8080      → overrides env
```

---

## One-Liner Parsing (Fail Fast)

If you prefer to panic on configuration errors and parse immediately:

```go
o := okapi.New()
cfg := &ServerConfig{}

cli := okapicli.New(o, "MyApp").
	FromStruct(cfg).
	MustParse() // Panics on error
```

This is useful for small tools or when invalid configuration should stop startup immediately.

---

## When to Use CLI vs Config Files

A common pattern is to combine CLI flags with configuration files:

* CLI → runtime overrides (ports, debug, mode)
* Config file → larger application configuration

Example:

```bash
./myapp --config=config.prod.yaml --port=9000 --debug
```

---

## Load Config File

You can load configuration from a file (YAML, JSON, etc.).

```go
o := okapi.New()
cfg := &Config{}

cli := okapicli.New(o, "MyApp")

if err := cli.LoadConfig("config.yaml", cfg); err != nil {
	panic(err)
}

```