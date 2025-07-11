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
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

// *********** SSE ***********

// Message represents a Server-Sent Events (SSE) message.
type Message struct {
	ID    string `json:"id" xml:"id"`
	Data  any    `json:"message"`
	Event string `json:"event"`
	Retry uint   `json:"retry,omitempty"` // Retry interval in milliseconds
}

// SendFunc defines the signature for a function that sends an SSE message.
type SendFunc func(data any, eventType string) (string, error)

// Send writes an SSE message to the response writer.
func (m *Message) Send(w http.ResponseWriter) (string, error) {
	setSSEHeaders(w)
	// Generate ID if not set
	if m.ID == "" {
		m.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	if err := writeID(w, m.ID); err != nil {
		return "", err
	}
	if err := writeEvent(w, m.Event); err != nil {
		return "", err
	}
	if err := writeRetry(w, m.Retry); err != nil {
		return "", err
	}
	if err := writeData(w, m.Data); err != nil {
		return "", err
	}

	flush(w)

	return m.ID, nil
}

// Close flushes the response writer to ensure data is sent to the client.
func (m *Message) Close(w http.ResponseWriter) error {
	flush(w)
	return nil
}

func flush(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func writeID(w http.ResponseWriter, id string) error {
	if id == "" {
		return nil
	}
	_, err := fmt.Fprintf(w, "id: %s\n", id)
	return err
}

func writeEvent(w http.ResponseWriter, eventType string) error {
	if eventType == "" {
		return nil
	}
	_, err := fmt.Fprintf(w, "event: %s\n", eventType)
	return err
}

func writeRetry(w http.ResponseWriter, retry uint) error {
	if retry <= 0 {
		return nil
	}
	_, err := fmt.Fprintf(w, "retry: %d\n", retry)
	return err
}

func writeData(w http.ResponseWriter, data any) error {
	var output string

	switch v := data.(type) {
	case string:
		output = v
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to encode data as JSON: %w", err)
		}
		output = string(jsonBytes)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}

func setSSEHeaders(w http.ResponseWriter) {
	header := w.Header()
	header["Content-Type"] = []string{"text/event-stream"}
	if _, ok := header["Cache-Control"]; !ok {
		header["Cache-Control"] = []string{"no-cache"}
	}

}
