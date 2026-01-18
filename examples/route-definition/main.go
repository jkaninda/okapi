package main

import (
	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/examples/route-definition/routes"
)

func main() {
	// Create a new Okapi instance with default config
	app := okapi.Default()
	// Create the route instance
	route := routes.NewRoute(app)
	route.RegisterRoutes()
	// Start the server
	if err := app.Start(); err != nil {
		panic(err)
	}

}
