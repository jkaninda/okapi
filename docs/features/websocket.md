---
title: WebSocket
layout: default
parent: Features
nav_order: 9
---
# WebSocket
Okapi provides WebSocket support through the `okapiws` package, enabling real-time, bidirectional communication between server and clients. This lightweight, framework-agnostic package offers a clean API and seamless integration with both Okapi and standard Go HTTP servers.


## Installation
```shell
go get github.com/jkaninda/okapi-ws
```

## Quick Start

### Basic Usage with Okapi

```go
package main

import (
	"log"
	"net/http"

	"github.com/jkaninda/okapi"
	okapiws "github.com/jkaninda/okapi-ws"
)

// WebSocket upgrades the HTTP connection to WebSocket.
// Config is optional; pass nil to use default settings.
func WebSocket(config *okapiws.WSConfig, c okapi.Context) (*okapiws.WSConnection, error) {
	upgrader := okapiws.NewWSUpgrader(config)
	return upgrader.Upgrade(c.Response(), c.Request(), nil)
}

// WebSocketWithHeaders upgrades with additional response headers.
func WebSocketWithHeaders(config *okapiws.WSConfig, headers http.Header, c okapi.Context) (*okapiws.WSConnection, error) {
	upgrader := okapiws.NewWSUpgrader(config)
	return upgrader.Upgrade(c.Response(), c.Request(), headers)
}

func main() {
	app := okapi.Default()

	app.Get("/", func(c okapi.Context) error {
		return c.OK(okapi.M{"message": "Hello from Okapi Web Framework!"})
	})

	app.Get("/ws", handleWebSocket)

	if err := app.Start(); err != nil {
		panic(err)
	}
}

func handleWebSocket(c *okapi.Context) error {
	ws, err := WebSocket(nil, c)
	if err != nil {
		return err
	}
	defer func() {
		if err := ws.Close(); err != nil {
			log.Printf("error closing WebSocket: %v", err)
		}
	}()

	ws.OnMessage(func(msg *okapiws.WSMessage) {
		log.Printf("[%d] %s", msg.Type, msg.Data)
		// Echo the message back
		_ = ws.Send(msg.Data)
	})

	ws.OnError(func(err error) {
		log.Printf("WebSocket error: %v", err)
	})

	ws.Start()

	// Block until the connection is closed
	<-ws.Context().Done()
	return nil
}
```
### Using With Go net/http

This example shows integration using Go’s standard library:

```go
package main

import (
	"log"
	"net/http"

	okapiws "github.com/jkaninda/okapi-ws"
)

func main() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		upgrader := okapiws.NewWSUpgrader(nil) // nil = use default config

		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
			return
		}
		defer ws.Close()

		ws.OnMessage(func(msg *okapiws.WSMessage) {
			log.Printf("[%d] %s", msg.Type, msg.Data)
			_ = ws.Send(msg.Data) // Echo
		})

		ws.OnError(func(err error) {
			log.Printf("WebSocket error: %v", err)
		})

		ws.Start()

		<-ws.Context().Done() // Block until closed
	})

	log.Println("Listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```


