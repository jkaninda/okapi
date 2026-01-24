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
	cli := okapicli.New(o, "Okapi CLI Example").
		String("config", "c", "", "Path to provider configuration file").
		Int("port", "p", 8000, "HTTP server port").
		Bool("debug", "d", false, "Enable debug mode")
	// Parse flags
	err := cli.ParseFlags()
	if err != nil {
		panic(err)
	}
	// Apply flag values to Okapi options
	o.WithPort(cli.GetInt("port"))
	if cli.GetBool("debug") {
		o.WithDebug()
	}
	config := &Config{}
	// Example of loading config from file
	configPath := cli.GetString("config")
	if configPath != "" {
		slog.Info("Loading configuration from file", "path", configPath)
		// Load
		if err = cli.LoadConfig(configPath, config); err != nil {
			// Panic on error or handle gracefully
			// panic(err)
			slog.Error("Failed to load configuration", "error", err)
		}
	}

	// Use flags
	o.Get("/", func(ctx *okapi.Context) error {
		return ctx.OK(okapi.M{
			"message": "Hello, Okapi!",
		})
	})

	if err = cli.RunServer(&okapicli.RunOptions{
		ShutdownTimeout: 30 * time.Second,                               // Optional: customize shutdown timeout
		Signals:         []os.Signal{okapicli.SIGINT, okapicli.SIGTERM}, // Optional: customize shutdown signals
		OnStart: func() {
			slog.Info("Ensuring resources are ready before starting...")
			if config != nil && config.DatabaseURL != "" {
				// This is just an example of using loaded config
				slog.Info("Connecting to database", "url", config.DatabaseURL)
			} else {
				slog.Error("No database URL provided in configuration")
			}
		},
		OnStarted: func() {
			slog.Info("Server started successfully")
			// You can add additional startup logic here
		},
		OnShutdown: func() {
			slog.Info("Cleanup before shutdown...")
			// Close database connections, etc.
		},
	}); err != nil {
		panic(err)
	}

	// Or simply use defaults
	// if err := cli.Run(); err != nil {
	//	panic(err)
	// }
}
