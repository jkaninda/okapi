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

package okapi

import (
	"net/http"
)

// *********** SSE ***********

// Message is a struct that represents a Server-Sent Events (SSE) connection.
type Message struct {
	// ID is the identifier for the SSE connection.
	ID string `json:"id" xml:"id"`
	// Data is the Data to be sent to the SSE connection.
	Data any `json:"message"`
	// Event is the event type for the SSE connection.
	Event string `json:"event"`
}
type Send func(data any, eventType string) (string, error)

// Send sends a message to the SSE connection.
func (s *Message) Send(w http.ResponseWriter, data any, eventType string) (string, error) {
	// Set the Content-Type header to text/event-stream
	w.Header().Set("Content-Type", "text/event-stream")
	// Set the Cache-Control header to no-cache
	w.Header().Set("Cache-Control", "no-cache")
	// Set the Connection header to keep-alive
	w.Header().Set("Connection", "keep-alive")

	// Write the SSE message to the response writer
	if _, err := w.Write([]byte("id: " + s.ID + "\n")); err != nil {
		return "", err
	}
	if _, err := w.Write([]byte("event: " + eventType + "\n")); err != nil {
		return "", err
	}
	if _, err := w.Write([]byte("data: " + data.(string) + "\n\n")); err != nil {
		return "", err
	}

	return s.ID, nil
}

// Close closes the SSE connection by flushing the response writer.
func (s *Message) Close(w http.ResponseWriter) error {
	// Flush the response writer to send the SSE message to the client
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}
