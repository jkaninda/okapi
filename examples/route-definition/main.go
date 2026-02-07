package main

import (
	"fmt"
	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/examples/route-definition/routes"
	"github.com/jkaninda/okapi/okapicli"
	"log"
	"os"
	"time"
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
