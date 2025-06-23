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
	// ************ Registering Routes ************
	app.Register(route.Home())
	app.Register(route.Version())
	app.Register(route.AdminRoutes()...)
	app.Register(route.AuthRoute())
	app.Register(route.SecurityRoutes()...)
	app.Register(route.BookRoutes()...)
	app.Register(route.V1BookRoutes()...)

	// Start the server
	if err := app.Start(); err != nil {
		panic(err)
	}

}
