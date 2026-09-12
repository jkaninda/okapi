package okapi

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// TestSSEFieldInjection covers id and event values derived from user input —
// a reflected Last-Event-ID being the obvious case. A newline in either would
// end the field and let the remainder be read as further SSE fields.
func TestSSEFieldInjection(t *testing.T) {
	rec := httptest.NewRecorder()
	m := &Message{
		ID:    "1\ndata: forged-by-attacker",
		Event: "ping\nevent: forged",
		Data:  "real",
	}
	if _, err := m.Send(rec); err != nil {
		t.Fatalf("Send: %v", err)
	}

	// Assert on the stream's structure: the payload survives as text inside
	// the id field, so a substring check would flag its own fix.
	body := rec.Body.String()
	var ids, events, data []string
	for _, line := range strings.Split(body, "\n") {
		switch {
		case strings.HasPrefix(line, "id:"):
			ids = append(ids, line)
		case strings.HasPrefix(line, "event:"):
			events = append(events, line)
		case strings.HasPrefix(line, "data:"):
			data = append(data, line)
		}
	}

	if len(ids) != 1 {
		t.Errorf("got %d id fields, want 1:\n%s", len(ids), body)
	}
	if len(events) != 1 {
		t.Errorf("got %d event fields, want 1:\n%s", len(events), body)
	}
	if len(data) != 1 {
		t.Errorf("got %d data fields, want 1:\n%s", len(data), body)
	}
	if len(data) == 1 && data[0] != "data: real" {
		t.Errorf("data field = %q, want %q", data[0], "data: real")
	}
}

// TestSSEDataLineEndings covers data containing CR or CRLF. The event stream
// treats a lone CR as a line end just like LF, so splitting data on LF alone
// let a CR in the payload start new fields and even a second event.
func TestSSEDataLineEndings(t *testing.T) {
	const payload = "hello\rid: evil\revent: admin\r\rdata: injected\r\nend"

	tests := []struct {
		name string
		msg  Message
	}{
		{"string data", Message{Data: payload}},
		{"byte data", Message{Data: []byte(payload)}},
		{"reader data", Message{Data: strings.NewReader(payload)}},
		{"serializer data", Message{Data: payload, Serializer: TextSerializer{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			msg := tt.msg
			msg.ID = "1"
			if _, err := msg.Send(rec); err != nil {
				t.Fatalf("Send: %v", err)
			}

			// Split the stream the way a client does: CRLF, CR and LF all end a line.
			body := rec.Body.String()
			lines := strings.Split(strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(body), "\n")
			want := []string{
				"id: 1",
				"data: hello",
				"data: id: evil",
				"data: event: admin",
				"data: ",
				"data: data: injected",
				"data: end",
				"",
				"",
			}
			if !reflect.DeepEqual(lines, want) {
				t.Errorf("stream lines = %q, want %q", lines, want)
			}
		})
	}
}

// TestBindDefaultBodyLimit covers the binders' default cap: BodyLimit is
// opt-in, so without one the default path read whatever a client sent.
func TestBindDefaultBodyLimit(t *testing.T) {
	type payload struct {
		Value string `json:"value"`
	}

	newApp := func(opts ...OptionFunc) *Okapi {
		app := New(opts...)
		app.Post("/bind", func(c *Context) error {
			var p payload
			if err := c.BindJSON(&p); err != nil {
				return c.String(http.StatusRequestEntityTooLarge, "too large")
			}
			return c.String(http.StatusOK, "ok")
		})
		return app
	}

	post := func(app *Okapi, size int) int {
		body := `{"value":"` + strings.Repeat("A", size) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/bind", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := post(newApp(), 1024); got != http.StatusOK {
		t.Errorf("small body: status = %d, want %d", got, http.StatusOK)
	}
	if got := post(newApp(), defaultMaxRequestBody+1024); got == http.StatusOK {
		t.Error("body over the default cap was accepted")
	}
	if got := post(newApp(WithMaxRequestBody(64<<20)), defaultMaxRequestBody+1024); got != http.StatusOK {
		t.Errorf("raised cap: status = %d, want %d", got, http.StatusOK)
	}
}
