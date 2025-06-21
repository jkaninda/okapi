package main

import (
	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/examples/route-definition/routes"
)

func main() {
	// Create a new Okapi instance with default config
	app := okapi.Default()
	// ************ Registering Routes ************
	// Register home route
	app.Register(routes.Home())
	// Register book routes
	app.Register(routes.BookRoutes()...)

	// Start the server
	if err := app.Start(); err != nil {
		panic(err)
	}
}
