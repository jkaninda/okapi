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
	"github.com/jkaninda/okapi/examples/route-definition/middlewares"
	"github.com/jkaninda/okapi/examples/route-definition/models"
	"net/http"
)

// ****************** Controllers ******************
var (
	bookController     = &controllers.BookController{}
	commonController   = &controllers.CommonController{}
	authController     = &controllers.AuthController{}
	bearerAuthSecurity = []map[string][]string{
		{
			"bearerAuth": {},
		},
	}
)

type Route struct {
	// app is the Okapi application
	app *okapi.Okapi
}

// NewRoute creates a new Route instance with the Okapi application
func NewRoute(app *okapi.Okapi) *Route {
	// Update OpenAPI documentation with the application title and version
	app.WithOpenAPIDocs(okapi.OpenAPI{
		Title:   "Okapi Web Framework Example",
		Version: "1.0.0",
		License: okapi.License{
			Name: "MIT",
		},
		SecuritySchemes: okapi.SecuritySchemes{
			{
				Name:   "basicAuth",
				Type:   "http",
				Scheme: "basic",
			},
			{
				Name:         "bearerAuth",
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
			{
				Name: "OAuth2",
				Type: "oauth2",
				Flows: &okapi.OAuthFlows{
					AuthorizationCode: &okapi.OAuthFlow{
						AuthorizationURL: "https://auth.example.com/authorize",
						TokenURL:         "https://auth.example.com/token",
						Scopes: map[string]string{
							"read":  "Read access",
							"write": "Write access",
						},
					},
				},
			},
		},
	})
	return &Route{
		app: app,
	}
}

// ****************** Routes Definition ******************

// Home returns the route definition for the Home endpoint
func (r *Route) Home() okapi.RouteDefinition {
	return okapi.RouteDefinition{
		Path:        "/",
		Method:      http.MethodGet,
		Handler:     commonController.Home,
		Group:       &okapi.Group{Prefix: "/", Tags: []string{"CommonController"}},
		Summary:     "Home Endpoint",
		Description: "This is the home endpoint of the Okapi Web Framework example application.",
	}
}

// Version returns the route definition for the Version endpoint
func (r *Route) Version() okapi.RouteDefinition {
	return okapi.RouteDefinition{
		Path:        "/version",
		Method:      http.MethodGet,
		Handler:     commonController.Version,
		Group:       &okapi.Group{Prefix: "/api/v1", Tags: []string{"CommonController"}},
		Summary:     "Version Endpoint",
		Description: "This endpoint returns the current version of the API.",
		Options: []okapi.RouteOption{
			okapi.DocResponse(okapi.M{"version": "v1"}),
		},
	}
}

// ************* Book Routes *************
// In this section, we will make BookRoutes deprecated and create BookV1Routes

// BookRoutes returns the route definitions for the BookController
func (r *Route) BookRoutes() []okapi.RouteDefinition {
	apiGroup := &okapi.Group{Prefix: "/api", Tags: []string{"BookController"}}
	// Mark the group as deprecated
	// But, it will still be available for use, it's just marked as deprecated on the OpenAPI documentation
	apiGroup.Deprecated()
	// Apply custom middleware
	apiGroup.Use(middlewares.CustomMiddleware)
	return []okapi.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/books",
			Handler:     bookController.GetBooks,
			Group:       apiGroup,
			Summary:     "Get Books",
			Description: "Retrieve a list of books",
			Options: []okapi.RouteOption{
				okapi.DocResponse(http.StatusBadRequest, models.ErrorResponse{}),
				okapi.DocResponse(http.StatusNotFound, models.ErrorResponse{}),
				okapi.DocResponse([]models.Book{}),
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/books/:id",
			Handler: bookController.GetBook,
			Group:   apiGroup,
			Options: []okapi.RouteOption{
				// OpenAPI Documentation can be added here or using the RouteDefinition fields directly
				okapi.DocSummary("Get Book by ID"),
				okapi.DocDescription("Retrieve a book by its ID"),
				okapi.DocPathParam("id", "int", "The ID of the book"),
				okapi.DocResponse(models.Book{}),
				okapi.DocResponse(http.StatusBadRequest, models.ErrorResponse{}),
				okapi.DocResponse(http.StatusNotFound, models.ErrorResponse{}),
			},
		},
	}
}

// *************** End of Book Routes ***************

// *********************** Book v1 Routes ***********************

func (r *Route) V1BookRoutes() []okapi.RouteDefinition {
	apiGroup := &okapi.Group{Prefix: "/api"}
	apiV1Group := apiGroup.Group("/v1").WithTags([]string{"BookController"})
	// Apply custom middleware
	// apiGroup.Use(middlewares.CustomMiddleware)
	return []okapi.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/books",
			Handler:     bookController.GetBooks,
			Group:       apiV1Group,
			Middlewares: []okapi.Middleware{middlewares.CustomMiddleware},
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
			Group:   apiV1Group,
			Options: []okapi.RouteOption{
				okapi.DocSummary("Get Book by ID"),
				okapi.DocDescription("Retrieve a book by its ID"),
				okapi.DocPathParam("id", "int", "The ID of the book"),
				okapi.DocResponse(models.Book{}),
				okapi.DocResponse(http.StatusBadRequest, models.ErrorResponse{}),
				okapi.DocResponse(http.StatusNotFound, models.ErrorResponse{}),
			},
		},
	}
}

// *************** Auth Routes ****************

// AuthRoute returns the route definition for the AuthController
func (r *Route) AuthRoute() okapi.RouteDefinition {
	// Create a new group for the AuthController
	apiGroup := &okapi.Group{Prefix: "/api/v1/auth", Tags: []string{"AuthController"}}
	// Apply custom middleware
	apiGroup.Use(middlewares.CustomMiddleware)
	return okapi.RouteDefinition{

		Method:  http.MethodPost,
		Path:    "/login",
		Handler: authController.Login,
		Group:   apiGroup,
		Options: []okapi.RouteOption{
			okapi.DocSummary("Login"),
			okapi.DocDescription("User login to get a JWT token"),
			okapi.DocRequestBody(models.AuthRequest{}),
			okapi.DocResponse(models.AuthResponse{}),
			okapi.DocResponse(http.StatusUnauthorized, models.AuthResponse{}),
		},
	}
}

// ************** Authenticated Routes **************

func (r *Route) SecurityRoutes() []okapi.RouteDefinition {
	coreGroup := &okapi.Group{Prefix: "/api/v1/security", Tags: []string{"SecurityController"}}
	// Apply JWT authentication middleware to the admin group
	coreGroup.Use(middlewares.JWTAuth.Middleware)
	// Apply custom middleware
	coreGroup.Use(middlewares.CustomMiddleware)
	return []okapi.RouteDefinition{
		{
			Method:  http.MethodPost,
			Path:    "/whoami",
			Handler: authController.WhoAmI,
			Group:   coreGroup,
			Options: []okapi.RouteOption{
				okapi.DocSummary("Whoami"),
				okapi.DocDescription("Get the current user's information"),
				okapi.DocResponse(models.UserInfo{}),
			},
			Security: bearerAuthSecurity,
		},
	}
}

// ***************** Admin Routes *****************

func (r *Route) AdminRoutes() []okapi.RouteDefinition {
	apiGroup := &okapi.Group{Prefix: "/api/v1/admin", Tags: []string{"AdminController"}}
	// Apply JWT authentication middleware to the admin group
	apiGroup.Use(middlewares.JWTAuth.Middleware)
	apiGroup.Use(middlewares.CustomMiddleware)
	apiGroup.WithSecurity(bearerAuthSecurity) // Apply Bearer token security to the group
	// apiGroup.WithBearerAuth() // Or you can use this to enable Bearer token for OpenAPI documentation

	return []okapi.RouteDefinition{

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
			// Security: bearerAuthSecurity, // Apply on the route level
		},
		{
			Method:  http.MethodDelete,
			Path:    "/books/:id",
			Handler: bookController.DeleteBook,
			Group:   apiGroup,
		},
	}
}

// ***************** End of Admin Routes *****************
