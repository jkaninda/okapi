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
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jkaninda/okapi/okapitest"
	"gopkg.in/yaml.v3"
)

const nameJane = "Jane"

type User struct {
	Name string `json:"name" required:"true" xml:"name" form:"name" query:"name" yaml:"name"`
}

type TestProduct struct {
	// String validation
	Name        string `json:"name" required:"true" minLength:"10" maxLength:"50"`
	Description string `json:"description" maxLength:"500"`
	SKU         string `json:"sku" required:"true" pattern:"^[A-Z]{3}-[0-9]{4}$"`

	// Number validation
	Price    float64 `json:"price" required:"true" min:"5" max:"100000"`
	Quantity int     `json:"quantity" min:"0" max:"1000" default:"1"`
	Discount float64 `json:"discount" min:"0" max:"100" multipleOf:"5"`

	// Enum validation
	Status   string `json:"status" required:"true" enum:"pending,paid,canceled,refunded"`
	Category string `json:"category" required:"true" enum:"electronics,clothing,books,food"`

	// Slice validation
	Tags     []string `json:"tags" minItems:"2" maxItems:"5" uniqueItems:"true"`
	Images   []string `json:"images" maxItems:"10"`
	Variants []string `json:"variants" uniqueItems:"true"`

	// Format validation
	SellerEmail string `json:"seller_email" required:"true" format:"email"`
	ProductID   string `json:"product_id" required:"true" format:"uuid"`
	SellerID    string `json:"seller_id" format:"uuid"`

	// Date/Time validation
	LaunchDate     string    `json:"launch_date" format:"date"`
	CreatedAt      time.Time `json:"created_at" required:"true" format:"date-time"`
	UpdatedAt      time.Time `json:"updated_at" format:"date-time"`
	ExpirationDate string    `json:"expiration_date" format:"date"`

	// Duration validation
	ShippingTime string `json:"shipping_time" format:"duration" default:"24h"`
	WarrantyTime string `json:"warranty_time" format:"duration"`

	// Network validation
	SellerWebsite string `json:"seller_website" format:"uri"`
	APIEndpoint   string `json:"api_endpoint" format:"uri"`
	SellerIP      string `json:"seller_ip" format:"ipv4"`
	ServerIPv6    string `json:"server_ipv6" format:"ipv6"`
	Hostname      string `json:"hostname" format:"hostname"`

	// Phone validation (regex)
	PhoneNumber string `json:"phone_number" format:"regex" pattern:"^\\+?[1-9]\\d{1,14}$"`

	// Combined validation
	Weight     float64   `json:"weight" required:"true" min:"0.1" max:"1000" multipleOf:"0.5"`
	Dimensions []float64 `json:"dimensions" minItems:"0" maxItems:"3"`
	Colors     []string  `json:"colors" minItems:"1" maxItems:"10" uniqueItems:"true"`

	// Default values
	IsActive bool   `json:"is_active" default:"true"`
	Currency string `json:"currency" default:"USD" enum:"USD,EUR,GBP,CDF"`
	Priority int    `json:"priority" default:"0" min:"0" max:"10"`
}

// validProductPayload returns a fresh map representing a fully valid TestProduct
// so callers can mutate fields without leaking into other test cases.
func validProductPayload() map[string]any {
	return map[string]any{
		"name":            "Valid Product Name",
		"description":     "A valid product description",
		"sku":             "ABC-1234",
		"price":           99.99,
		"quantity":        10,
		"discount":        10.0,
		"status":          "pending",
		"category":        "electronics",
		"tags":            []string{"tag1", "tag2", "tag3"},
		"images":          []string{"image1.jpg", "image2.jpg"},
		"variants":        []string{"red", "blue"},
		"seller_email":    "seller@example.com",
		"product_id":      "123e4567-e89b-12d3-a456-426614174000",
		"seller_id":       "123e4567-e89b-12d3-a456-426614174001",
		"launch_date":     "2024-12-08",
		"created_at":      time.Now().Format(time.RFC3339),
		"updated_at":      time.Now().Format(time.RFC3339),
		"expiration_date": "2025-12-08",
		"shipping_time":   "24h",
		"warranty_time":   "3600h30ms",
		"seller_website":  "https://example.com",
		"api_endpoint":    "https://api.example.com",
		"seller_ip":       "192.168.1.1",
		"server_ipv6":     "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		"hostname":        "example.com",
		"phone_number":    "+243999999999",
		"weight":          10.5,
		"dimensions":      []float64{10.0, 20.0, 30.0},
		"colors":          []string{"red", "blue", "green"},
		"is_active":       true,
		"currency":        "USD",
		"priority":        5,
	}
}

// bindJSON marshals payload as JSON and binds it into out using ctx.Bind.
// It returns the error from Bind so callers can assert on it.
func bindJSON(t *testing.T, payload any, out any) error {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	ctx, _ := NewTestContext(http.MethodPost, "/test", bytes.NewReader(body))
	ctx.Request().Header.Set("Content-Type", "application/json")
	return ctx.Bind(out)
}

func TestContext_Bind(t *testing.T) {
	ts := NewTestServer(t)

	ts.Get("/", func(c *Context) error { return c.XML(http.StatusOK, books) })
	ts.Get("/hello", func(c *Context) error { return c.Text(http.StatusOK, "Hello World!") })

	// Bind() + ShouldBind() composite: success and failure paths both exercised.
	ts.Post("/hello", func(c *Context) error {
		u := User{}
		if err := c.Bind(&u); err != nil {
			return c.AbortBadRequest("Bad requests")
		}
		if ok, err := c.ShouldBind(&u); !ok {
			return c.AbortBadRequest("Bad requests", err)
		}
		return c.JSON(http.StatusCreated, u)
	})

	// B: thin wrapper around Bind.
	ts.Put("/hello", func(c *Context) error {
		u := User{}
		if err := c.B(&u); err != nil {
			return c.AbortBadRequest("Bad requests")
		}
		return c.JSON(http.StatusCreated, u)
	})

	ts.Post("/json", func(c *Context) error {
		u := User{}
		if err := c.BindJSON(&u); err != nil {
			return c.AbortBadRequest("Bad requests", err)
		}
		return c.JSON(http.StatusCreated, u)
	})
	ts.Post("/xml", func(c *Context) error {
		u := User{}
		if err := c.BindXML(&u); err != nil {
			return c.AbortBadRequest("Bad requests", err)
		}
		return c.JSON(http.StatusCreated, u)
	})
	ts.Post("/yaml", func(c *Context) error {
		u := User{}
		if err := c.BindYAML(&u); err != nil {
			return c.ErrorBadRequest("Bad requests")
		}
		return c.JSON(http.StatusCreated, u)
	})
	// /form and /query use the unified c.Bind path, which routes form-encoded
	// POSTs and query-only GETs through bindFromFields (per-tag string setter).
	// BindForm/BindQuery directly are exercised separately below — they decode
	// url.Values via JSON and therefore only accept []string fields.
	ts.Post("/form", func(c *Context) error {
		u := User{}
		if err := c.Bind(&u); err != nil {
			return c.AbortBadRequest("Bad requests", err)
		}
		return c.JSON(http.StatusCreated, u)
	})
	ts.Get("/query", func(c *Context) error {
		u := User{}
		if err := c.Bind(&u); err != nil {
			return c.AbortBadRequest("Bad requests", err)
		}
		return c.JSON(http.StatusOK, u)
	})
	ts.Post("/multipart", func(c *Context) error {
		u := User{}
		if err := c.BindMultipart(&u); err != nil {
			return c.AbortBadRequest("Bad requests", err)
		}
		return c.JSON(http.StatusCreated, u)
	})
	// proto.Unmarshal panics on a nil message, so we cannot exercise
	// BindProtoBuf via a route handler. Direct assertions on the no-server
	// failure path live in TestBindProtoBuf_NilTarget below.

	t.Run("root XML returns 200", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/").ExpectStatusOK()
	})

	t.Run("Bind fails on empty POST body", func(t *testing.T) {
		okapitest.POST(t, ts.BaseURL+"/hello").ExpectStatusBadRequest()
	})

	t.Run("BindJSON fails on empty POST body", func(t *testing.T) {
		okapitest.POST(t, ts.BaseURL+"/json").ExpectStatusBadRequest()
	})

	t.Run("BindJSON decodes valid body", func(t *testing.T) {
		okapitest.POST(t, ts.BaseURL+"/json").
			JSONBody(map[string]string{"name": nameJane}).
			ExpectStatus(http.StatusCreated).
			ExpectBodyContains(`"name":"Jane"`)
	})

	t.Run("BindXML decodes valid body", func(t *testing.T) {
		body, err := xml.Marshal(User{Name: nameJane})
		if err != nil {
			t.Fatalf("marshal xml: %v", err)
		}
		okapitest.POST(t, ts.BaseURL+"/xml").
			Header("Content-Type", "application/xml").
			Body(bytes.NewReader(body)).
			ExpectStatus(http.StatusCreated).
			ExpectBodyContains(`"name":"Jane"`)
	})

	t.Run("BindYAML decodes valid body", func(t *testing.T) {
		body, err := yaml.Marshal(User{Name: nameJane})
		if err != nil {
			t.Fatalf("marshal yaml: %v", err)
		}
		okapitest.POST(t, ts.BaseURL+"/yaml").
			Header("Content-Type", "application/yaml").
			Body(bytes.NewReader(body)).
			ExpectStatus(http.StatusCreated).
			ExpectBodyContains(`"name":"Jane"`)
	})

	t.Run("BindForm decodes valid body", func(t *testing.T) {
		okapitest.POST(t, ts.BaseURL+"/form").
			FormBody(map[string]string{"name": nameJane}).
			ExpectStatus(http.StatusCreated).
			ExpectBodyContains(`"name":"Jane"`)
	})

	t.Run("BindQuery decodes valid query", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/query").
			QueryParam("name", nameJane).
			ExpectStatusOK().
			ExpectBodyContains(`"name":"Jane"`)
	})

	t.Run("BindMultipart decodes valid form-data", func(t *testing.T) {
		body, ct := buildMultipart(t, map[string]string{"name": nameJane})
		okapitest.POST(t, ts.BaseURL+"/multipart").
			Header("Content-Type", ct).
			Body(body).
			ExpectStatus(http.StatusCreated).
			ExpectBodyContains(`"name":"Jane"`)
	})

	t.Run("B wraps Bind", func(t *testing.T) {
		okapitest.PUT(t, ts.BaseURL+"/hello").
			JSONBody(map[string]string{"name": nameJane}).
			ExpectStatus(http.StatusCreated).
			ExpectBodyContains(`"name":"Jane"`)
	})
}

func buildMultipart(t *testing.T, fields map[string]string) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("write field %q: %v", k, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return &buf, w.FormDataContentType()
}

func TestBind_ValidProduct(t *testing.T) {
	t.Parallel()

	var product TestProduct
	if err := bindJSON(t, validProductPayload(), &product); err != nil {
		t.Fatalf("bind valid product: %v", err)
	}

	if product.Name != "Valid Product Name" {
		t.Errorf("Name = %q, want %q", product.Name, "Valid Product Name")
	}
	if product.Price != 99.99 {
		t.Errorf("Price = %v, want %v", product.Price, 99.99)
	}
	if product.Status != "pending" {
		t.Errorf("Status = %q, want %q", product.Status, "pending")
	}
	if product.Category != "electronics" {
		t.Errorf("Category = %q, want %q", product.Category, "electronics")
	}
	if len(product.Tags) != 3 {
		t.Errorf("Tags length = %d, want 3", len(product.Tags))
	}
}

func TestBind_RequiredFields(t *testing.T) {
	t.Parallel()

	requiredFields := []string{"name", "price", "seller_email", "sku", "status", "category", "product_id", "created_at", "weight"}

	for _, f := range requiredFields {
		t.Run("missing "+f, func(t *testing.T) {
			t.Parallel()

			payload := validProductPayload()
			delete(payload, f)

			var product TestProduct
			if err := bindJSON(t, payload, &product); err == nil {
				t.Errorf("expected error when %q is missing, got nil", f)
			}
		})
	}
}

func TestBind_StringValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{"name too short", "name", "Short", true},
		{"name valid length", "name", "Valid Product Name", false},
		{"name too long", "name", strings.Repeat("a", 51), true},
		{"description too long", "description", strings.Repeat("d", 501), true},
		{"SKU lowercase rejected", "sku", "abc-1234", true},
		{"SKU wrong shape rejected", "sku", "AB-1234", true},
		{"SKU valid", "sku", "ABC-1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := validProductPayload()
			payload[tt.field] = tt.value

			var product TestProduct
			err := bindJSON(t, payload, &product)
			if tt.wantError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBind_NumberValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		price     float64
		quantity  int
		discount  float64
		weight    float64
		wantError bool
	}{
		{"price below minimum", 2.0, 10, 10, 10.5, true},
		{"price above maximum", 200000.0, 10, 10, 10.5, true},
		{"quantity below minimum", 99.99, -1, 10, 10.5, true},
		{"quantity above maximum", 99.99, 1001, 10, 10.5, true},
		{"discount not multiple of 5", 99.99, 10, 7, 10.5, true},
		{"discount valid multiple", 99.99, 10, 15, 10.5, false},
		{"weight not multiple of 0.5", 99.99, 10, 10, 10.3, true},
		{"weight valid multiple", 99.99, 10, 10, 10.5, false},
		{"all values valid", 99.99, 10, 10, 10.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := validProductPayload()
			payload["price"] = tt.price
			payload["quantity"] = tt.quantity
			payload["discount"] = tt.discount
			payload["weight"] = tt.weight

			var product TestProduct
			err := bindJSON(t, payload, &product)
			if tt.wantError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBind_EnumValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    string
		category  string
		currency  string
		wantError bool
	}{
		{"invalid status value", "invalid", "electronics", "USD", true},
		{"valid status pending", "pending", "electronics", "USD", false},
		{"invalid category value", "pending", "invalid", "USD", true},
		{"valid category clothing", "paid", "clothing", "EUR", false},
		{"invalid currency value", "pending", "electronics", "INVALID", true},
		{"valid currency CDF", "canceled", "books", "CDF", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := validProductPayload()
			payload["status"] = tt.status
			payload["category"] = tt.category
			payload["currency"] = tt.currency

			var product TestProduct
			err := bindJSON(t, payload, &product)
			if tt.wantError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBind_SliceValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tags      []string
		images    []string
		colors    []string
		wantError bool
	}{
		{"tags too few items", []string{"tag1"}, []string{"img1"}, []string{"red"}, true},
		{"tags too many items", []string{"t1", "t2", "t3", "t4", "t5", "t6"}, []string{"img1"}, []string{"red"}, true},
		{"tags not unique", []string{"tag1", "tag1", "tag2"}, []string{"img1"}, []string{"red"}, true},
		{"images too many", []string{"t1", "t2"}, []string{"i1", "i2", "i3", "i4", "i5", "i6", "i7", "i8", "i9", "i10", "i11"}, []string{"red"}, true},
		{"colors too few", []string{"t1", "t2"}, []string{"img1"}, []string{}, true},
		{"colors not unique", []string{"t1", "t2"}, []string{"img1"}, []string{"red", "red"}, true},
		{"all slices valid", []string{"t1", "t2"}, []string{"img1", "img2"}, []string{"red", "blue"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := validProductPayload()
			payload["tags"] = tt.tags
			payload["images"] = tt.images
			payload["colors"] = tt.colors

			var product TestProduct
			err := bindJSON(t, payload, &product)
			if tt.wantError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBind_FormatValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{"invalid email", "seller_email", "invalid-email", true},
		{"valid email", "seller_email", "valid@example.com", false},
		{"invalid uuid", "product_id", "not-a-uuid", true},
		{"valid uuid", "product_id", "123e4567-e89b-12d3-a456-426614174000", false},
		{"invalid date", "launch_date", "2024-13-45", true},
		{"valid date", "launch_date", "2024-12-08", false},
		{"invalid duration", "shipping_time", "invalid", true},
		{"valid duration", "shipping_time", "24h30m", false},
		{"invalid uri", "seller_website", "not a uri", true},
		{"valid uri", "seller_website", "https://example.com", false},
		{"invalid ipv4", "seller_ip", "999.999.999.999", true},
		{"valid ipv4", "seller_ip", "192.168.1.1", false},
		{"invalid ipv6", "server_ipv6", "invalid", true},
		{"valid ipv6", "server_ipv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"invalid hostname", "hostname", "invalid_hostname!", true},
		{"valid hostname", "hostname", "example.com", false},
		{"invalid phone regex", "phone_number", "abc", true},
		{"valid phone regex", "phone_number", "+243999999999", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := validProductPayload()
			payload[tt.field] = tt.value

			var product TestProduct
			err := bindJSON(t, payload, &product)
			if tt.wantError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBind_DefaultValues(t *testing.T) {
	t.Parallel()

	// Strip every field that has a default tag so we observe the defaults
	// being applied rather than overwritten by the input.
	payload := validProductPayload()
	delete(payload, "quantity")
	delete(payload, "shipping_time")
	delete(payload, "is_active")
	delete(payload, "currency")
	delete(payload, "priority")

	var product TestProduct
	if err := bindJSON(t, payload, &product); err != nil {
		t.Fatalf("bind: %v", err)
	}

	if product.Quantity != 1 {
		t.Errorf("Quantity default = %d, want 1", product.Quantity)
	}
	if product.ShippingTime != "24h" {
		t.Errorf("ShippingTime default = %q, want %q", product.ShippingTime, "24h")
	}
	if !product.IsActive {
		t.Errorf("IsActive default = false, want true")
	}
	if product.Currency != "USD" {
		t.Errorf("Currency default = %q, want %q", product.Currency, "USD")
	}
	if product.Priority != 0 {
		t.Errorf("Priority default = %d, want 0", product.Priority)
	}
}

func TestBind_MalformedBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{"invalid JSON syntax", `{invalid json}`},
		{"empty object", `{}`}, // missing required fields
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, _ := NewTestContext(http.MethodPost, "/test", strings.NewReader(tt.body))
			ctx.Request().Header.Set("Content-Type", "application/json")

			var product TestProduct
			if err := ctx.Bind(&product); err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestBind_InvalidTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target any
	}{
		{"non-pointer", User{}},
		{"nil pointer", (*User)(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, _ := NewTestContext(http.MethodPost, "/test", strings.NewReader(`{"name":"Jane"}`))
			ctx.Request().Header.Set("Content-Type", "application/json")

			if err := ctx.Bind(tt.target); err == nil {
				t.Errorf("expected error for %s target, got nil", tt.name)
			}
		})
	}
}

type formTarget struct {
	Name []string `json:"name"`
}

func TestBindForm_DecodesValues(t *testing.T) {
	t.Parallel()

	form := url.Values{"name": []string{nameJane}}
	ctx, _ := NewTestContext(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	ctx.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var got formTarget
	if err := ctx.BindForm(&got); err != nil {
		t.Fatalf("BindForm: %v", err)
	}
	if len(got.Name) != 1 || got.Name[0] != nameJane {
		t.Errorf("Name = %v, want [Jane]", got.Name)
	}
}

func TestBindQuery_DecodesValues(t *testing.T) {
	t.Parallel()

	ctx, _ := NewTestContext(http.MethodGet, "/test?name=Jane", nil)

	var got formTarget
	if err := ctx.BindQuery(&got); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}
	if len(got.Name) != 1 || got.Name[0] != nameJane {
		t.Errorf("Name = %v, want [Jane]", got.Name)
	}
}
