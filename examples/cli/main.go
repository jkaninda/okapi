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
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/okapicli"
)

type ServerConfig struct {
	Port   int    `cli:"port"   short:"p" desc:"HTTP server port"        env:"APP_PORT"   default:"8080"`
	Debug  bool   `cli:"debug"  short:"d" desc:"Enable debug mode"      env:"APP_DEBUG"`
	Config string `cli:"config" short:"c" desc:"Path to config file"   default:"config.yaml"`
}

type Config struct {
	DatabaseURL  string  `yaml:"database_url"`
	SecretKey    string  `yaml:"secret_key"`
	AllowedHosts string  `yaml:"allowed_hosts"`
	Workers      int     `yaml:"workers"`
	Timeout      float64 `yaml:"timeout"`
}

func main() {
	o := okapi.Default()
	cli := okapicli.New(o, "myapp")

	//  Subcommand: serve
	serveCfg := &ServerConfig{}
	cli.Command("serve", "Start the HTTP server", func(cmd *okapicli.Command) error {
		// Apply parsed config
		cmd.Okapi().WithPort(serveCfg.Port)
		if serveCfg.Debug {
			cmd.Okapi().WithDebug()
		}

		// Load app config from file
		config := &Config{}
		if path := serveCfg.Config; path != "" {
			slog.Info("Loading configuration", "path", path)
			if err := cmd.CLI().LoadConfig(path, config); err != nil {
				slog.Error("Failed to load configuration", "error", err)
			}
		}

		// Define routes
		cmd.Okapi().Get("/", func(ctx *okapi.Context) error {
			return ctx.OK(okapi.M{"message": "Hello, Okapi!"})
		})

		// Run server with lifecycle hooks
		return cmd.CLI().RunServer(&okapicli.RunOptions{
			ShutdownTimeout: 30 * time.Second,
			Signals:         []os.Signal{okapicli.SIGINT, okapicli.SIGTERM},
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
		})
	}).FromStruct(serveCfg)

	// Subcommand: worker
	cli.Command("worker", "Start application worker", func(cmd *okapicli.Command) error {

		fmt.Println("Starting myapp wortker...")
		sleepTime := cmd.GetDuration("sleep") * time.Second
		slog.Info("Worker started successfully", "concurrency", cmd.GetInt("concurrency"), "sleep", sleepTime)

		time.Sleep(sleepTime)
		fmt.Println("Stoping myapp wortker...")
		return nil
	}).Int("concurrency", "c", 3, "concurrency jobs").Duration("sleep", "", 5, "Sleep time")
	// Subcommand: version
	cli.Command("version", "Show the worker", func(cmd *okapicli.Command) error {
		fmt.Println("myapp v1.0.0")
		return nil
	})

	// Execute: parses os.Args for the subcommand and runs it
	// Usage:
	//   myapp serve --port 9090 --debug
	//   myapp worker --concurrency 5 --slep 5
	//   myapp version
	if err := cli.Execute(); err != nil {
		slog.Error("Error", "error", err)
		os.Exit(1)
	}
}
