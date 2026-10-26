---
title: CLI
sidebar_position: 10
---

# CLI Integration

Okapi comes with a built-in command-line interface (CLI) helper that makes it easy to configure and run your application from the terminal.

It supports:

- Typed flags (`string`, `int`, `bool`, `float`, `time.Duration`)
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
if debug {
	app.WithDebug() // WithDebug takes no argument; call it only when debug is on
}

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
```

`RunServer` starts the server, waits for a signal, then shuts down with
`StopWithContext`, which drains in-flight requests before cancelling their
contexts. Fields left at their zero value keep their defaults: `ShutdownTimeout`
is 30 seconds and `Signals` is `SIGINT`, `SIGTERM`. The signal handler is released
when `RunServer` returns, including when the server fails to start — so a process
whose listener could not bind still terminates on `SIGINT`.

`cli.Run()` is a shortcut for `RunServer(nil)`, and `cli.RunServer()` with no
argument behaves the same.

### Flag Methods

| Define                                       | Read                    |
|----------------------------------------------|-------------------------|
| `String(name, short, def string, usage)`     | `GetString(name)`       |
| `Int(name, short, def int, usage)`           | `GetInt(name)`          |
| `Bool(name, short, def bool, usage)`         | `GetBool(name)`         |
| `Float(name, short, def float64, usage)`     | `GetFloat(name)`        |
| `Duration(name, short, def time.Duration, usage)` | `GetDuration(name)` |

Pass `""` as `short` for a flag without a shorthand. A getter called with an
unknown name, or with a name of another type, returns the zero value.

```go
cli := okapicli.New(app, "myapp").
	Float("rate", "r", 1.5, "Requests per second").
	Duration("timeout", "t", 30*time.Second, "Request timeout")

if err := cli.Parse(); err != nil {
	panic(err)
}

rate := cli.GetFloat("rate")
timeout := cli.GetDuration("timeout") // --timeout=5s
```

`Parse` is an alias for `ParseFlags`. Both apply `env` bindings first, then parse
`os.Args[1:]`. A malformed flag, or `--help`, prints the usage and exits the process.

`okapicli.Default()` creates a CLI around `okapi.Default()`, and `cli.Okapi()`
returns the wrapped instance. `okapicli.SIGINT` and `okapicli.SIGTERM` are
provided for `RunOptions.Signals`.

### Example CLI Output

```bash
$ ./myapp --help

Usage of myapp:
  -c, --config string   Path to configuration file (default "config.yaml")
  -d, --debug           Enable debug mode
  -p, --port int        HTTP server port (default 8080)
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
	Host   string `cli:"host"             desc:"Server hostname"   env:"APP_HOST"   default:"localhost"`
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

Supported field types are `string`, the `int` kinds, `bool`, the `float` kinds and
`time.Duration` (its `default` tag is parsed with `time.ParseDuration`, e.g.
`default:"30s"`); fields of other types are skipped.

Avoid `short:"h"`: it shadows the `-h` shorthand that would otherwise print the
help text. `--help` keeps working either way.

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

After parsing, the final values are written back into the struct, so `cfg.Port` and
`cli.GetInt("port")` agree. One exception: on the top-level CLI, a `time.Duration`
field gets its flag but its parsed value is not written back — read it with
`cli.GetDuration(name)`. Commands (below) write durations back as well.

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

`LoadConfig` reads a `.json`, `.yaml` or `.yml` file into a struct. Any other
extension is rejected with an error.

```go
o := okapi.New()
cfg := &Config{}

cli := okapicli.New(o, "MyApp")

if err := cli.LoadConfig("config.yaml", cfg); err != nil {
	panic(err)
}
```

---

## Subcommands

`Command` registers a subcommand with its own flags and a handler; `Execute`
picks the command from the first non-flag argument, parses that command's flags,
and runs it:

```go
cli := okapicli.New(app, "myapp")

cli.Command("serve", "Start the HTTP server", func(cmd *okapicli.Command) error {
	cmd.Okapi().WithPort(cmd.GetInt("port"))
	return cmd.CLI().RunServer()
}).Int("port", "p", 8080, "HTTP server port")

cli.Command("migrate", "Run migrations", func(cmd *okapicli.Command) error {
	steps := cmd.Args() // positional arguments after the flags
	_ = steps
	return nil
}).Bool("dry-run", "", false, "Print the migrations without running them")

cli.DefaultCommand("serve")

if err := cli.Execute(); err != nil {
	log.Fatal(err)
}
```

```bash
./myapp serve -p 9000
./myapp migrate --dry-run 001 002
./myapp -p 9000          # no command: runs the default command, "serve"
```

- A `*Command` has the same flag methods as the CLI (`String`, `Int`, `Bool`,
  `Float`, `Duration` and their `Get*` readers), plus `FromStruct(&cfg)` with the
  same tags, `Args()` for the remaining positional arguments, `Name()`, `CLI()` and
  `Okapi()`.
- When no argument names a registered command, `Execute` runs the
  `DefaultCommand` with all the arguments. Without a default command it prints
  the list of commands and returns an error.
- Once commands are registered, `Execute` parses only the matched command's
  flags — flags defined on the CLI itself are not parsed. Without any command,
  `Execute` behaves like `ParseFlags`.
- `cli.MatchedCommand()` returns the command `Execute` ran, or `nil`.

---

## Export the OpenAPI Document

`WithOpenAPICommand` registers an `openapi` command that writes the
application's OpenAPI document without starting the server, for a client
generator or to check it in:

```go
cli := okapicli.New(o, "MyApp").WithOpenAPICommand()
cli.Command("serve", "Start the server", func(cmd *okapicli.Command) error {
    return cli.RunServer()
})
cli.DefaultCommand("serve")
registerRoutes(o) // before Execute

if err := cli.Execute(); err != nil {
    log.Fatal(err)
}
```

```bash
./myapp openapi                     # OpenAPI 3.1 JSON on stdout
./myapp openapi -o openapi.yaml     # format taken from the extension
./myapp openapi --version 3.0 -f yaml
```

| Flag              | Default  | Description                                   |
|-------------------|----------|-----------------------------------------------|
| `-o`, `--output`  | stdout   | File to write the document to                 |
| `-f`, `--format`  | `json`   | `json` or `yaml`; inferred from `--output`    |
| `-v`, `--version` | `3.1`    | OpenAPI version, `3.1` or `3.0`               |
