/*
 *  MIT License
 *
 * Copyright (c) 2026 Jonas Kaninda
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

package main

import (
	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/okapicli"
	"log/slog"
	"os"
	"time"
)

type Config struct {
	DatabaseURL  string  `yaml:"database_url"`
	SecretKey    string  `yaml:"secret_key"`
	AllowedHosts string  `yaml:"allowed_hosts"`
	Workers      int     `yaml:"workers"`
	Timeout      float64 `yaml:"timeout"`
}

func main() {
	// Create default Okapi instance
	o := okapi.Default()

	// Create CLI instance
	// Or cli := okapicli.New(o) // The name is optional
	cli := okapicli.New(o, "Okapi CLI Example").
		String("config", "c", "config.yaml", "Path to configuration file").
		Int("port", "p", 8000, "HTTP server port").
		Bool("debug", "d", false, "Enable debug mode")

	// Parse flags
	if err := cli.ParseFlags(); err != nil {
		panic(err)
	}

	// Apply CLI options
	o.WithPort(cli.GetInt("port"))
	if cli.GetBool("debug") {
		o.WithDebug()
	}

	// Load configuration
	config := &Config{}
	if path := cli.GetString("config"); path != "" {
		slog.Info("Loading configuration", "path", path)
		if err := cli.LoadConfig(path, config); err != nil {
			slog.Error("Failed to load configuration", "error", err)
		}
	}

	// Define routes
	o.Get("/", func(ctx *okapi.Context) error {
		return ctx.OK(okapi.M{
			"message": "Hello, Okapi!",
		})
	})

	// Run server with lifecycle hooks
	if err := cli.RunServer(&okapicli.RunOptions{
		ShutdownTimeout: 30 * time.Second,                               // Optional: customize shutdown timeout
		Signals:         []os.Signal{okapicli.SIGINT, okapicli.SIGTERM}, // Optional: customize shutdown signals
		OnStart: func() {
			slog.Info("Preparing resources before startup")
			if config.DatabaseURL != "" {
				slog.Info("Connecting to database")
			}
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

	// Or use defaults
	// if err := cli.Run(); err != nil {
	//	panic(err)
	// }
}
