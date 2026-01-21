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
	goutils "github.com/jkaninda/go-utils"
	"github.com/jkaninda/okapi"
	"net/http"
	"time"
)

const template = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .name }}</title>
    <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
    <style>
        body {
            font-family: Arial, sans-serif;
            padding: 24px;
            color: #222;
        }

        h1 {
            margin-bottom: 4px;
            font-size: 22px;
        }

        p {
            margin-bottom: 16px;
            color: #555;
        }

        .status {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 20px;
            font-size: 14px;
        }

        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: #28a745;
        }

        .data-grid {
            display: grid;
            gap: 12px;
            max-width: 350px;
        }

        .data-item {
            padding: 8px 12px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }

        .data-label {
            font-size: 11px;
            text-transform: uppercase;
            color: #666;
            margin-bottom: 2px;
        }

        .data-value {
            font-size: 18px;
            font-weight: 600;
            font-family: monospace;
        }
    </style>
</head>
<body x-data="sseHandler()">
    <h1>{{ .name }}</h1>
    <p>{{ .message }}</p>

    <div class="status">
        <span class="status-dot" x-show="connected"></span>
        <span x-text="connected ? 'Connected' : 'Disconnected'"></span>
    </div>

    <div class="data-grid">
        <div class="data-item">
            <div class="data-label">Server Time</div>
            <div class="data-value" x-text="time || '--:--:--'"></div>
        </div>

        <div class="data-item">
            <div class="data-label">Connected Since</div>
            <div class="data-value" x-text="connectedSince || '--'"></div>
        </div>
    </div>

    <script>
        function sseHandler() {
            return {
                time: null,
                connectedSince: null,
                connected: false,
                eventSource: null,

                init() {
                    this.connect();
                },

                connect() {
                    this.eventSource = new EventSource('{{ .eventURL }}');

                    this.eventSource.addEventListener('message', (event) => {
                        const data = JSON.parse(event.data);
                        this.time = data.time;
                        this.connectedSince = data.connectedSince;
                        this.connected = true;
                    });

                    this.eventSource.onerror = () => {
                        this.connected = false;
                        this.eventSource.close();
                        setTimeout(() => this.connect(), 3000);
                    };
                }
            }
        }
    </script>
</body>
</html>
`

func main() {
	// Example usage of SSE handling in Okapi
	//
	// Creates a new Okapi instance
	o := okapi.Default()

	o.Get("/", func(c *okapi.Context) error {
		return c.HTMLView(http.StatusOK, template, okapi.M{
			"name":     "OKAPI GO Web Framework",
			"message":  "This is an example of SSE",
			"eventURL": "/events",
		})
	})
	o.Get("/events", func(c *okapi.Context) error {
		// Simulate sending events (you can replace this with real data)
		connectedAt := time.Now()
		// Send events every second
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		ctx := c.Request().Context()
		for {
			select {
			case <-ctx.Done():
				return nil
			case t := <-ticker.C:
				data := okapi.M{
					"time":           t.Format("15:04:05"),
					"connectedSince": goutils.FormatDuration(time.Since(connectedAt), 1),
					"timestamp":      t.Unix(),
				}

				if err := c.SSEvent("message", data); err != nil {
					return err
				}
			}
		}
	})

	// Start the server
	err := o.Start()
	if err != nil {
		panic(err)
	}
}
