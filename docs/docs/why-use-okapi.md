---
title: Why Use Okapi?
sidebar_position: 4
---

# Why Use Okapi?

**Okapi** is a modern, high-performance web framework for building RESTful APIs in Go. It combines simplicity, speed, and flexibility to help you create robust web services with ease.

## Key Features

* **Easy to Learn**: With familiar Go syntax and intuitive APIs, you can be productive in minutes—even on your first project.
* **Lightweight and Unopinionated**: Okapi is built from the ground up and doesn't wrap or build on top of another framework. It gives you full control without unnecessary abstraction or bloat.
* **Highly Flexible**: Designed to adapt to your architecture and workflow—not the other way around.
* **Built for Production**: Fast, reliable, and efficient under real-world load. Okapi is optimized for performance without sacrificing developer experience.
* **Standard Library Compatibility**: Integrates seamlessly with Go's net/http standard library, making it easy to combine Okapi with existing Go code and tools.
* **Automatic OpenAPI Documentation**: Generate comprehensive OpenAPI specs automatically for every route — OpenAPI 3.1 by default, with the same description also published as 3.0 — keeping your API documentation in sync with your code.
* **Dynamic Route Management**: Enable or disable routes and route groups without commenting out code. A disabled route or group answers `404`, and for groups the decision is taken on every request, so it also covers routes registered before the call. The OpenAPI document is built when the server starts, so toggling afterwards changes routing rather than the published document.
* **Extensible Middleware System**: Build reusable middleware components that can be shared across routes and projects.
* **Robust Request Binding**: Automatically parse and validate incoming requests (JSON, XML, YAML, Protobuf, form data, query params, headers, cookies, path variables) into Go structs with minimal boilerplate — with decode errors reported and request bodies capped at 8 MB by default.
* **Comprehensive Tooling**: Includes features like route grouping, static file serving, single-page-application hosting, templating engine support, CORS management, and fine-grained timeout controls.
* **Route Definition & Middleware Chaining**: Organize your routes and middleware in a clean, modular way for better maintainability.

## Ideal for

* **High-performance REST APIs**
* **Composable microservices**
* **Rapid prototyping**
* **Learning & teaching Go web development**

Whether you're building your next startup, internal tools, or side projects, **Okapi scales with you.**
