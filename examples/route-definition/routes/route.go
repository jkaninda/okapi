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

package routes

import (
	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/examples/route-definition/controllers"
	"github.com/jkaninda/okapi/examples/route-definition/models"
	"net/http"
)

var (
	bookController = &controllers.BookController{}
	homeController = &controllers.HomeController{}
)

// You can also use this example

// type Route struct {
//	app *okapi.Okapi
// }
// // NewRoute creates a new Route instance with the provided Okapi app
// func NewRoute(app *okapi.Okapi) *Route {
//	return &Route{
//		app: app,
//	}
// }
// func (r *Route) Home()okapi.RouteDefinition  {
//	// you can access directly the okapi app instance
//	// r.app.Get("/", homeController.Home)
//
//	// or use okapi.RouteDefinition
//	return okapi.RouteDefinition{
//		Path:    "/",
//		Method:  http.MethodGet,
//		Handler: homeController.Home,
//		Group:   &okapi.Group{Prefix: "/", Tags: []string{"HomeController"}},
//	}
// }
// // in main.go, you can register the routes like this:
// app := okapi.Default()
// route := routes.NewRoute(app)
// // Register the home route
// app.Register(route.Home())

// ****************** Route Definitions ******************

// Home returns the route definition for the HomeController
func Home() okapi.RouteDefinition {
	return okapi.RouteDefinition{
		Path:    "/",
		Method:  http.MethodGet,
		Handler: homeController.Home,
		Group:   &okapi.Group{Prefix: "/", Tags: []string{"HomeController"}},
	}
}

// ************* Book Routes *************

// BookRoutes returns the route definitions for the BookController
func BookRoutes() []okapi.RouteDefinition {
	apiGroup := &okapi.Group{Prefix: "/api", Tags: []string{"BookController"}}
	return []okapi.RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/books",
			Handler: bookController.GetBooks,
			Group:   apiGroup,
			Options: []okapi.RouteOption{
				okapi.DocSummary("Get Books"),
				okapi.DocDescription("Retrieve a list of books"),
				okapi.DocResponse([]models.Book{}),
				okapi.DocResponse(http.StatusBadRequest, models.ErrorResponse{}),
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/books/:id",
			Handler: bookController.GetBook,
			Group:   apiGroup,
			Options: []okapi.RouteOption{
				okapi.DocSummary("Get Book by ID"),
				okapi.DocDescription("Retrieve a book by its ID"),
				okapi.DocPathParam("id", "int", "The ID of the book"),
				okapi.DocResponse(models.Book{}),
				okapi.DocResponse(http.StatusBadRequest, models.ErrorResponse{}),
				okapi.DocResponse(http.StatusNotFound, models.ErrorResponse{}),
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/books",
			Handler: bookController.CreateBook,
			Group:   apiGroup,
			Options: []okapi.RouteOption{
				okapi.DocSummary("Create Book"),
				okapi.DocDescription("Create a new book"),
				okapi.DocRequestBody(models.Book{}),
				okapi.DocResponse(models.Response{}),
			},
		},
	}
}

// You can add more controllers here as needed, e.g.:
