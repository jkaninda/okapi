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

// Package okapi is a modern, minimalist HTTP web framework for Go,
// inspired by FastAPI's elegance. Designed for simplicity, performance,
// and developer happiness, it helps you build fast, scalable, and well-documented APIs
// with minimal boilerplate.
//
// The framework is named after the okapi (/oʊˈkɑːpiː/), a rare and graceful mammal
// native to the rainforests of the northeastern Democratic Republic of the Congo.
// Just like its namesake — which resembles a blend of giraffe and zebra — Okapi blends
// simplicity and strength in a unique, powerful package.
//
// Key Features:
//
//   - Intuitive & Expressive API:
//     Clean, declarative syntax for effortless route and middleware definition.
//
//   - Automatic Request Binding:
//     Seamlessly parse JSON, XML, form data, query params, headers, and path variables into structs.
//
//   - Built-in Auth & Security:
//     Native support for JWT, OAuth2, Basic Auth, and custom middleware.
//
//   - Blazing Fast Routing:
//     Optimized HTTP router with low overhead for high-performance applications.
//
//   - First-Class Documentation:
//     OpenAPI 3.0 & Swagger UI integrated out of the box—auto-generate API docs with minimal effort.
//
//   - Modern Tooling:
//     Route grouping, middleware chaining, static file serving, templating engine support,
//     CORS management, fine-grained timeout controls.
//
//   - Developer Experience:
//     Minimal boilerplate, clear error handling, structured logging, and easy testing.
//
// Okapi is built for speed, simplicity, and real-world use—whether you're prototyping or running in production.
//
// For more information and documentation, visit: https://github.com/jkaninda/okapi
package okapi
