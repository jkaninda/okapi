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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"time"
)

// *********** SSE ***********

// Message represents a Server-Sent Events (SSE) message.
type Message struct {
	// ID unique identifier for the message
	ID string `json:"id"`
	// Event type
	Event string `json:"event"`
	//  Data payload
	Data any `json:"message"`
	// Retry interval
	Retry uint `json:"retry,omitempty"`
	// Serializer to use for the message
	Serializer Serializer `json:"-"`
}
type StreamOptions struct {
	// Serializer to use for the stream messages
	Serializer Serializer
	// PingEnabled indicates whether to send periodic ping messages to keep the connection alive.
	PingInterval time.Duration
	// OnError is a callback function to handle errors during streaming.
	OnError func(error)
}

// Serializer defines how to convert data to string format
//
//	Implement this interface to create custom serializers for SSE messages.
type Serializer interface {
	Serialize(data any) (string, error)
}

// JSONSerializer is the default JSON serializer
type JSONSerializer struct{}

func (j JSONSerializer) Serialize(data any) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to encode data as JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// TextSerializer for plain text/string data
type TextSerializer struct{}

func (t TextSerializer) Serialize(data any) (string, error) {
	return fmt.Sprintf("%v", data), nil
}

// Base64Serializer for binary data
type Base64Serializer struct{}

func (b Base64Serializer) Serialize(data any) (string, error) {
	switch v := data.(type) {
	case []byte:
		return base64.StdEncoding.EncodeToString(v), nil
	default:
		return "", fmt.Errorf("base64 serializer requires []byte data")
	}
}

// SendFunc defines the signature for a function that sends an SSE message.
type SendFunc func(data any, eventType string) (string, error)

// Send writes an SSE message to the response writer.
func (m *Message) Send(w http.ResponseWriter) (string, error) {
	m.setSSEHeaders(w)
	// Generate ID if not set
	if m.ID == "" {
		m.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	if err := m.writeID(w, m.ID); err != nil {
		return "", err
	}
	if err := m.writeEvent(w, m.Event); err != nil {
		return "", err
	}
	if err := m.writeRetry(w, m.Retry); err != nil {
		return "", err
	}
	if err := m.writeData(w, m.Data); err != nil {
		return "", err
	}

	m.flush(w)

	return m.ID, nil
}

// Close flushes the response writer to ensure data is sent to the client.
func (m *Message) Close(w http.ResponseWriter) error {
	m.flush(w)
	return nil
}

func (m *Message) flush(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (m *Message) writeID(w http.ResponseWriter, id string) error {
	if id == "" {
		return nil
	}
	_, err := fmt.Fprintf(w, "id: %s\n", id)
	return err
}

func (m *Message) writeEvent(w http.ResponseWriter, eventType string) error {
	if eventType == "" {
		return nil
	}
	_, err := fmt.Fprintf(w, "event: %s\n", eventType)
	return err
}

func (m *Message) writeRetry(w http.ResponseWriter, retry uint) error {
	if retry <= 0 {
		return nil
	}
	_, err := fmt.Fprintf(w, "retry: %d\n", retry)
	return err
}

func (m *Message) writeData(w http.ResponseWriter, data any) error {
	var output string
	var err error
	if data == nil {
		_, err = fmt.Fprint(w, "data: \n\n")
		return err
	}

	// Use custom serializer if provided
	if m.Serializer != nil {
		output, err = m.Serializer.Serialize(data)
		if err != nil {
			return err
		}
	} else {
		// Default behavior
		switch v := data.(type) {
		case string:
			output = v
		case []byte:
			output = string(v)
		case io.Reader:
			// stream data
			buf := new(strings.Builder)
			if _, err := io.Copy(buf, v); err != nil {
				return fmt.Errorf("failed to read data: %w", err)
			}
			output = buf.String()
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to encode data as JSON: %w", err)
			}
			output = string(jsonBytes)
		}
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if _, err = fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(w, "\n")
	return err
}

func (m *Message) setSSEHeaders(w http.ResponseWriter) {
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Connection", "keep-alive")
	if _, ok := header["Cache-Control"]; !ok {
		header.Set("Cache-Control", "no-cache")
	}
}
