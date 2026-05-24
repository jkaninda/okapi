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

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/examples/route-definition/routes"
	"github.com/jkaninda/okapi/okapicli"
)

func main() {
	// Create a new Okapi instance with default config
	app := okapi.Default()
	// Create a new CLI manager for the Okapi instance
	cli := okapicli.New(app, "Router Definition Example").
		Int("port", "p", 8000, "HTTP server port").
		Bool("debug", "d", false, "Enable debug mode")

	// Parse CLI flags
	if err := cli.Parse(); err != nil {
		log.Fatal("Failed to parse CLI flags", "error", err)
	}
	// Apply CLI options to the Okapi instance
	app.WithPort(cli.GetInt("port"))
	if cli.GetBool("debug") {
		app.WithDebug()
	}
	// Create the router instance
	router := routes.NewRouter(app)
	router.RegisterRoutes()
	// Start the server
	if err := cli.RunServer(&okapicli.RunOptions{
		ShutdownTimeout: 30 * time.Second,                               // Optional: customize shutdown timeout
		Signals:         []os.Signal{okapicli.SIGINT, okapicli.SIGTERM}, // Optional: customize shutdown signals
		OnStart: func() {
			fmt.Println("Preparing resources before startup")

		},
		OnStarted: func() {
			fmt.Println("Server started successfully")
		},
		OnShutdown: func() {
			fmt.Println("Cleaning up before shutdown")
		},
	}); err != nil {
		log.Fatal("Failed to start server", "error", err)

	}
}
