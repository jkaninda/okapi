---
title: CLI
layout: default
parent: Features
nav_order: 10
---

# CLI

Okapi includes a command-line interface (CLI) tool to help you quickly run and manage your Okapi applications.

Example usage:

```go
	app := okapi.New()
	// Create CLI instance
	cli := okapicli.New(app, "Goma").
		String("config", "c", "config.yaml", "Path to configuration file").
		Int("port", "p", 8080, "HTTP server port").
		Bool("debug", "d", false, "Enable debug mode")
		
    // Parse flags
	if err := cli.ParseFlags(); err != nil {
		panic(err)
	}
    // Retrieve flag values
    port := cli.GetInt("port")
    app.WithPort(port)
       
	app.Get("/", func(ctx *okapi.Context) error {
	return ctx.OK(okapi.M{
			"status":  "ok",
			"message": "CLI example",
		})
  })
  
  // Run server
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