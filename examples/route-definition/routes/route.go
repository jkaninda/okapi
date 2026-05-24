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
	"net/http"

	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/examples/route-definition/handlers"
	"github.com/jkaninda/okapi/examples/route-definition/middlewares"
	"github.com/jkaninda/okapi/examples/route-definition/models"
)

// ****************** Handlers ******************
var (
	bookHandler        = &handlers.BookHandler{}
	commonHandler      = &handlers.CommonHandler{}
	authHandler        = &handlers.AuthHandler{}
	bearerAuthSecurity = []map[string][]string{
		{
			"bearerAuth": {},
		},
	}
)

type Router struct {
	// app is the Okapi application
	app   *okapi.Okapi
	group *okapi.Group
}

// NewRouter creates a new Router instance with the Okapi application
func NewRouter(app *okapi.Okapi) *Router {
	// Update OpenAPI documentation with the application title and version
	app.WithOpenAPIDocs(okapi.OpenAPI{
		Title:       "Okapi Web Framework Example",
		Version:     "1.0.0",
		Description: "Okapi Web Framework Route Definition Example",
		Summary:     "Okapi Web Framework Route Definition Example",
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
	app.WithDocUI(okapi.ScalarUI)
	return &Router{
		app:   app,
		group: &okapi.Group{Prefix: "/api/v1"},
	}
}

// ************ Registering Routes ************

func (r *Router) RegisterRoutes() {
	// Register all routes
	r.app.Register(r.home())
	r.app.Register(r.version())
	r.app.Register(r.bookRoutes()...)
	r.app.Register(r.v1BookRoutes()...)
	r.app.Register(r.authRoute())
	r.app.Register(r.securityRoutes()...)
	r.app.Register(r.adminRoutes()...)

}

// ****************** Routes Definition ******************

// home returns the route definition for the Home endpoint
func (r *Router) home() okapi.RouteDefinition {
	return okapi.RouteDefinition{
		Path:        "/",
		Method:      http.MethodGet,
		Handler:     commonHandler.Home,
		Group:       &okapi.Group{Prefix: "/", Tags: []string{"Common"}},
		Summary:     "Home Endpoint",
		Description: "This is the home endpoint of the Okapi Web Framework example application.",
	}
}

// version returns the route definition for the version endpoint
func (r *Router) version() okapi.RouteDefinition {
	return okapi.RouteDefinition{
		Path:        "/version",
		Method:      http.MethodGet,
		Handler:     commonHandler.Version,
		Group:       &okapi.Group{Prefix: "/api/v1", Tags: []string{"Common"}},
		Summary:     "version Endpoint",
		Description: "This endpoint returns the current version of the API.",
		Options: []okapi.RouteOption{
			okapi.DocResponse(okapi.M{"version": "v1"}),
		},
	}
}

// ************* Book Routes *************
// In this section, we will make bookRoutes deprecated and create BookV1Routes

// bookRoutes returns the route definitions for the BookController
func (r *Router) bookRoutes() []okapi.RouteDefinition {
	apiGroup := &okapi.Group{Prefix: "/api", Tags: []string{"Books"}}
	// Mark the group as deprecated
	// But, it will still be available for use, it's just marked as deprecated on the OpenAPI documentation
	apiGroup.Deprecated()
	// Apply custom middleware
	apiGroup.Use(middlewares.CustomMiddleware)
	return []okapi.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/books",
			Handler:     bookHandler.List,
			Group:       apiGroup,
			Summary:     "Get Books",
			Description: "Retrieve a list of books",
			Options: []okapi.RouteOption{
				okapi.DocResponse(http.StatusBadRequest, &models.ErrorResponseDto{}),
				okapi.DocResponse(http.StatusNotFound, &models.ErrorResponseDto{}),
				okapi.DocResponse(&models.BooksResponse{}),
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/books/:id",
			Handler: bookHandler.Get,
			Group:   apiGroup,
			Options: []okapi.RouteOption{
				// OpenAPI Documentation can be added here or using the RouteDefinition fields directly
				okapi.DocSummary("Get Book by ID"),
				okapi.DocDescription("Retrieve a book by its ID"),
				okapi.DocPathParam("id", "int", "The ID of the book"),
				okapi.DocResponse(models.Book{}),
				okapi.DocResponse(http.StatusBadRequest, &models.ErrorResponseDto{}),
				okapi.DocResponse(http.StatusNotFound, &models.ErrorResponseDto{}),
			},
		},
	}
}

// *************** End of Book Routes ***************

// *********************** Book v1 Routes ***********************

func (r *Router) v1BookRoutes() []okapi.RouteDefinition {
	apiGroup := r.group.Group("/books").WithTags([]string{"V1Books"})
	// Apply custom middleware
	// apiGroup.Use(middlewares.CustomMiddleware)
	return []okapi.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/",
			Handler:     bookHandler.List,
			Group:       apiGroup,
			Middlewares: []okapi.Middleware{middlewares.CustomMiddleware},
			Summary:     "Get Books",
			Description: "Retrieve a list of books",
			Response:    &models.BooksResponse{},
			Options: []okapi.RouteOption{
				okapi.DocResponse(http.StatusBadRequest, models.ErrorResponseDto{}),
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/{id:int}",
			Handler: bookHandler.Get,
			Group:   apiGroup,
			Options: []okapi.RouteOption{
				okapi.DocSummary("Get Book by ID"),
				okapi.DocDescription("Retrieve a book by its ID"),
				okapi.DocResponse(models.Book{}),
				okapi.DocResponse(http.StatusBadRequest, &models.ErrorResponseDto{}),
				okapi.DocResponse(http.StatusNotFound, &models.ErrorResponseDto{}),
			},
		},
	}
}

// *************** Auth Routes ****************

// authRoute returns the route definition for the AuthController
func (r *Router) authRoute() okapi.RouteDefinition {
	// Create a new group for the AuthController
	apiGroup := r.group.Group("/auth").WithTags([]string{"Auth"})

	// Apply custom middleware
	apiGroup.Use(middlewares.CustomMiddleware)
	return okapi.RouteDefinition{
		Method:      http.MethodPost,
		Path:        "/login",
		Handler:     authHandler.Login,
		Group:       apiGroup,
		Summary:     "Login",
		Description: "User login to get a JWT token",
		Request:     &models.AuthRequest{},
		Response:    &models.ResponseDto[models.AuthResponse]{},
		Options: []okapi.RouteOption{
			okapi.DocResponse(http.StatusUnauthorized, models.AuthResponse{}),
		},
	}
}

// ************** Authenticated Routes **************

func (r *Router) securityRoutes() []okapi.RouteDefinition {
	coreGroup := r.group.Group("/security").WithTags([]string{"Security"})
	coreGroup.Use(middlewares.JWTAuth.Middleware)
	// Apply custom middleware
	coreGroup.Use(middlewares.CustomMiddleware)
	return []okapi.RouteDefinition{
		{
			Method:  http.MethodPost,
			Path:    "/whoami",
			Handler: authHandler.WhoAmI,
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

func (r *Router) adminRoutes() []okapi.RouteDefinition {
	apiGroup := r.group.Group("/admin").WithTags([]string{"Admin"})
	// Apply JWT authentication middleware to the admin group
	apiGroup.Use(middlewares.JWTAuth.Middleware)
	apiGroup.Use(middlewares.CustomMiddleware)
	apiGroup.WithSecurity(bearerAuthSecurity) // Apply Bearer token security to the group
	// apiGroup.WithBearerAuth() // Or you can use this to enable Bearer token for OpenAPI documentation

	return []okapi.RouteDefinition{

		{
			Method:      http.MethodPost,
			Path:        "/books",
			Handler:     bookHandler.Create,
			Group:       apiGroup,
			Summary:     "Create a new book",
			Description: "Create a new book in the system",
			Request:     &models.BookRequest{},
			Response:    &models.ResponseDto[models.Book]{},
			// Security: bearerAuthSecurity, // Apply on the route level
		},
		{
			Method:      http.MethodGet,
			Path:        "/books",
			Handler:     bookHandler.List,
			Group:       apiGroup,
			Summary:     "Get Books",
			Description: "Retrieve a list of books",
			Options: []okapi.RouteOption{
				okapi.DocResponse(http.StatusBadRequest, &models.ErrorResponseDto{}),
				okapi.DocResponse([]models.Book{}),
			},
			// Security: bearerAuthSecurity, // Apply on the route level
		},
		{
			Method:      http.MethodPut,
			Path:        "/books/{id:int}",
			Handler:     bookHandler.Update,
			Group:       apiGroup,
			Summary:     "Update a book",
			Description: "Update a book",
			Request:     &models.BookUpdateRequest{},
			Response:    &models.BookResponse{},
			// Security: bearerAuthSecurity, // Apply on the route level
			Options: []okapi.RouteOption{
				okapi.DocResponse(http.StatusBadRequest, &models.ErrorResponseDto{}),
			},
		},
		{
			Method:      http.MethodDelete,
			Path:        "/books/{id:int}",
			Handler:     bookHandler.Delete,
			Group:       apiGroup,
			OperationId: "deleteBook",
			Summary:     "Delete a book",
			Description: "Delete a book",
		},
	}
}

// ***************** End of Admin Routes *****************
