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

package main

import (
	"github.com/jkaninda/okapi"
	"net/http"
	"time"
)

const template = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>OKAPI SSE Example</title>
</head>
<body>
<h1>{{.name}}</h1>
<p>{{.message}}</p>
<div id="sse-data"></div>
<script>
  const eventSource = new EventSource({{.eventURL}});
  eventSource.onmessage = function(event) {
    const dataElement = document.getElementById('sse-data');
    dataElement.innerHTML += event.data + '<br>';
  };
</script>
</body>`

func main() {
	// Example usage of SSE handling in Okapi
	// Create a new Okapi instance
	o := okapi.Default()

	o.Get("/", func(c okapi.Context) error {
		return c.HTMLView(http.StatusOK, template, okapi.M{
			"name":     "OKAPI",
			"message":  "This is an example of SSE",
			"eventURL": "http://localhost:8080/events",
		})
	})
	o.Get("/events", func(c okapi.Context) error {
		// Simulate sending events (you can replace this with real data)
		for i := 0; i < 10; i++ {
			
			data := okapi.M{"name": "Okapi", "License": "MIT", "event": "SSE example"}
			event := "message"

			err := c.SSEvent(event, data)
			if err != nil {
				return err
			}
			time.Sleep(2 * time.Second)
		}
		return nil
	})

	// Start the server
	err := o.Start()
	if err != nil {
		panic(err)
	}
}
