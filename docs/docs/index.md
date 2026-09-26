---
title: Overview
slug: /
sidebar_position: 1
---

# Introduction

**Okapi** is a modern, minimalist HTTP framework for Go built for **simplicity, performance, and developer experience**. It helps you build fast, scalable, and well-documented APIs with minimal boilerplate.

Designed around **clarity, strong typing, and developer productivity**, Okapi provides automatic request validation and built-in OpenAPI documentation out of the box, making it easy to create production-ready APIs that are clean, maintainable, and easy to evolve.

Named after the okapi (/oʊˈkɑːpiː/), a rare and graceful mammal native to the rainforests of the Democratic Republic of the Congo, the framework reflects the same balance of elegance, adaptability, and strength. Like the animal itself, Okapi combines distinct qualities into a unique and powerful experience.

![Okapi Logo](https://raw.githubusercontent.com/jkaninda/okapi/main/logo.png)

## ✨ Key Features

### 🚀 Fast & Minimal

- Lightweight and optimized for high-performance HTTP applications
- Minimal boilerplate with an intuitive developer-friendly API
- Fully compatible with Go’s `net/http` standard library

### 🧩 Powerful Request Handling

- Automatic binding for:
  - JSON
  - XML
  - YAML
  - Protobuf
  - Form data
  - Query parameters
  - Headers
  - Cookies
  - Path parameters
- Strongly typed request handling with built-in validation
- Request bodies are capped (8 MB by default, adjustable with `WithMaxRequestBody`)

### 🔐 Security & Middleware

- Native support for:
  - JWT authentication (HS256, RS256, JWKS)
  - Basic Authentication
  - Custom middleware
- Route grouping and middleware chaining
- Fine-grained timeout controls, including a default `ReadHeaderTimeout`
- CORS management

### 📚 Built-in API Documentation

- Automatic OpenAPI generation: **3.1 by default**, with the same description also served as **3.0**
- Three interactive UIs bundled: **Swagger UI** (default), **ReDoc**, and **Scalar**
- Documentation is generated from your route definitions, so it stays in sync with your code

### ⚡ Dynamic Route Management

- Enable or disable routes and route groups without commenting out code
- A disabled route or group answers `404 Not Found`; for groups the decision is
  re-evaluated on every request, so it also covers routes registered earlier
- Routes and groups disabled before the server starts are left out of the
  generated OpenAPI document (the document is built at startup, so toggling
  afterwards changes routing but not the published document)

### 🛠 Modern Developer Experience

- Clear and consistent error handling
- Structured logging support
- Easy testing and maintainable architecture
- Static file serving and single-page-application hosting
- Templating engine support (HTML-escaping by default)

Built for **real-world applications**, Okapi scales from quick prototypes to production-grade services with a focus on speed, clarity, and maintainability.

---

## Badges

[![Tests](https://github.com/jkaninda/okapi/actions/workflows/tests.yml/badge.svg)](https://github.com/jkaninda/okapi/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jkaninda/okapi)](https://goreportcard.com/report/github.com/jkaninda/okapi)
[![Go Reference](https://pkg.go.dev/badge/github.com/jkaninda/okapi.svg)](https://pkg.go.dev/github.com/jkaninda/okapi)
[![codecov](https://codecov.io/gh/jkaninda/okapi/branch/main/graph/badge.svg?token=JHTW49M1LF)](https://codecov.io/gh/jkaninda/okapi)
[![GitHub Release](https://img.shields.io/github/v/release/jkaninda/okapi)](https://github.com/jkaninda/okapi/releases)
