---
title: Installation
layout: home
nav_order: 2
---
# Installation

## Prerequisites

- Go 1.21 or higher

## Create a New Project

```bash
mkdir myapi && cd myapi
go mod init myapi
```

## Install Okapi

```bash
go get github.com/jkaninda/okapi@latest
```

That's it! You're ready to start building your API with Okapi.

## Verify Installation

Create a simple `main.go` file to verify the installation:

```go
package main

import (
    "github.com/jkaninda/okapi"
)

func main() {
    o := okapi.Default()
    
    o.Get("/", func(c *okapi.Context) error {
        return c.OK(okapi.M{"message": "Hello from Okapi!"})
    })
    
    if err := o.Start(); err != nil {
        panic(err)
    }
}
```

Run the server:

```bash
go run main.go
```

Visit `http://localhost:8080` to see your API in action!
