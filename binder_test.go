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
	"errors"
	"github.com/jkaninda/okapi/okapitest"
	"net/http"
	"testing"
	"time"
)

type User struct {
	Name string `json:"name" required:"true" xml:"name" form:"name" query:"name" yaml:"name"`
}

func TestContext_Bind(t *testing.T) {
	o := Default()

	o.Get("/", func(c *Context) error {
		return c.XML(http.StatusOK, books)
	})
	o.Get("/hello", func(c *Context) error {
		return c.Text(http.StatusOK, "Hello World!")
	})
	o.Post("/hello", func(c *Context) error {
		user := User{}
		if err := c.Bind(&user); err != nil {
			return c.AbortBadRequest("Bad requests")
		}
		if ok, err := c.ShouldBind(&user); !ok {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Put("/hello", func(c *Context) error {
		user := User{}
		if err := c.B(&user); err != nil {
			return c.AbortBadRequest("Bad requests")
		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Get("/hello", func(c *Context) error {
		return c.JSON(http.StatusOK, books)
	})

	o.Post("/bind", func(c *Context) error {
		user := User{}
		if ok, err := c.ShouldBind(&user); !ok {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/multipart", func(c *Context) error {
		user := User{}
		if err := c.BindMultipart(&user); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/xml", func(c *Context) error {
		user := User{}
		if err := c.BindXML(&user); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/form", func(c *Context) error {
		user := User{}
		if err := c.BindForm(&user); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Get("/query", func(c *Context) error {
		user := User{}
		if err := c.BindQuery(&user); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/json", func(c *Context) error {
		user := User{}
		if err := c.BindJSON(&user); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/yaml", func(c *Context) error {
		user := User{}
		if err := c.BindYAML(&user); err != nil {
			return c.ErrorBadRequest("Bad requests")

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/protobuf", func(c *Context) error {
		if err := c.BindProtoBuf(nil); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.OK(http.StatusOK)
	})
	// Start server in background
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err := o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()

	okapitest.GET(t, "http://localhost:8080").ExpectStatusOK()
	okapitest.POST(t, "http://localhost:8080/hello").ExpectStatusBadRequest()
	okapitest.POST(t, "http://localhost:8080/json").ExpectStatusBadRequest()

}

// TestProduct demonstrates comprehensive validation tags
type TestProduct struct {
	// Basic string validation
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

func TestBind_ValidProduct(t *testing.T) {
	validProduct := map[string]interface{}{
		"name":            "Test Product Name",
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

	jsonBody, _ := json.Marshal(validProduct)
	body := bytes.NewBuffer(jsonBody)
	ctx, rec := NewTestContext(http.MethodPost, "/test", body)
	ctx.Request().Header.Set("Content-Type", "application/json")

	var product TestProduct
	if err := ctx.Bind(&product); err != nil {
		t.Errorf("Expected no error for valid product, got %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("Expected response code 200, got %d", rec.Code)
	}

	// Verify some fields
	if product.Name != "Test Product Name" {
		t.Errorf("Expected name 'Test Product Name', got '%s'", product.Name)
	}
	if product.Price != 99.99 {
		t.Errorf("Expected price 99.99, got %f", product.Price)
	}
	if product.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", product.Status)
	}
}

func TestBind_RequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		payload     map[string]interface{}
		expectError bool
		description string
	}{
		{
			name: "missing required name",
			payload: map[string]interface{}{
				"sku":          "ABC-1234",
				"price":        99.99,
				"status":       "pending",
				"category":     "electronics",
				"tags":         []string{"tag1", "tag2"},
				"seller_email": "seller@example.com",
				"product_id":   "123e4567-e89b-12d3-a456-426614174000",
				"created_at":   time.Now().Format(time.RFC3339),
				"weight":       10.5,
				"colors":       []string{"red"},
			},
			expectError: true,
			description: "Name is required",
		},
		{
			name: "missing required price",
			payload: map[string]interface{}{
				"name":         "Test Product Name",
				"sku":          "ABC-1234",
				"status":       "pending",
				"category":     "electronics",
				"tags":         []string{"tag1", "tag2"},
				"seller_email": "seller@example.com",
				"product_id":   "123e4567-e89b-12d3-a456-426614174000",
				"created_at":   time.Now().Format(time.RFC3339),
				"weight":       10.5,
				"colors":       []string{"red"},
			},
			expectError: true,
			description: "Price is required",
		},
		{
			name: "missing required seller_email",
			payload: map[string]interface{}{
				"name":       "Test Product Name",
				"sku":        "ABC-1234",
				"price":      99.99,
				"status":     "pending",
				"category":   "electronics",
				"tags":       []string{"tag1", "tag2"},
				"product_id": "123e4567-e89b-12d3-a456-426614174000",
				"created_at": time.Now().Format(time.RFC3339),
				"weight":     10.5,
				"colors":     []string{"red"},
			},
			expectError: true,
			description: "Seller email is required",
		},
		{
			name: "missing required SKU",
			payload: map[string]interface{}{
				"name":         "Test Product Name",
				"price":        99.99,
				"status":       "pending",
				"category":     "electronics",
				"tags":         []string{"tag1", "tag2"},
				"seller_email": "seller@example.com",
				"product_id":   "123e4567-e89b-12d3-a456-426614174000",
				"created_at":   time.Now().Format(time.RFC3339),
				"weight":       10.5,
				"colors":       []string{"red"},
			},
			expectError: true,
			description: "SKU is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.payload)
			body := bytes.NewBuffer(jsonBody)
			ctx, _ := NewTestContext(http.MethodPost, "/test", body)
			ctx.Request().Header.Set("Content-Type", "application/json")

			var product TestProduct
			err := ctx.Bind(&product)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestBind_StringValidation(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		value       string
		expectError bool
		description string
	}{
		{
			name:        "name too short",
			fieldName:   "name",
			value:       "Short",
			expectError: true,
			description: "Name must be at least 10 characters",
		},
		{
			name:        "name valid length",
			fieldName:   "name",
			value:       "Valid Product Name",
			expectError: false,
			description: "Name with valid length",
		},
		{
			name:        "name too long",
			fieldName:   "name",
			value:       "This is a very long product name that exceeds the fifty character limit for sure",
			expectError: true,
			description: "Name must not exceed 50 characters",
		},
		{
			name:        "description too long",
			fieldName:   "description",
			value:       string(make([]byte, 501)),
			expectError: true,
			description: "Description must not exceed 500 characters",
		},
		{
			name:        "invalid SKU pattern lowercase",
			fieldName:   "sku",
			value:       "abc-1234",
			expectError: true,
			description: "SKU must match pattern ABC-1234",
		},
		{
			name:        "invalid SKU pattern wrong format",
			fieldName:   "sku",
			value:       "AB-1234",
			expectError: true,
			description: "SKU must be 3 uppercase letters, dash, 4 digits",
		},
		{
			name:        "valid SKU pattern",
			fieldName:   "sku",
			value:       "ABC-1234",
			expectError: false,
			description: "Valid SKU format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"name":         "Valid Product Name",
				"sku":          "ABC-1234",
				"price":        99.99,
				"status":       "pending",
				"category":     "electronics",
				"tags":         []string{"tag1", "tag2"},
				"seller_email": "seller@example.com",
				"product_id":   "123e4567-e89b-12d3-a456-426614174000",
				"created_at":   time.Now().Format(time.RFC3339),
				"weight":       10.5,
				"colors":       []string{"red"},
			}
			payload[tt.fieldName] = tt.value

			jsonBody, _ := json.Marshal(payload)
			body := bytes.NewBuffer(jsonBody)
			ctx, _ := NewTestContext(http.MethodPost, "/test", body)
			ctx.Request().Header.Set("Content-Type", "application/json")

			var product TestProduct
			err := ctx.Bind(&product)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestBind_NumberValidation(t *testing.T) {
	tests := []struct {
		name        string
		price       float64
		quantity    int
		discount    float64
		weight      float64
		expectError bool
		description string
	}{
		{
			name:        "price below minimum",
			price:       2.0,
			quantity:    10,
			discount:    10,
			weight:      10.5,
			expectError: true,
			description: "Price must be at least 5",
		},
		{
			name:        "price above maximum",
			price:       200000.0,
			quantity:    10,
			discount:    10,
			weight:      10.5,
			expectError: true,
			description: "Price must not exceed 100000",
		},
		{
			name:        "quantity below minimum",
			price:       99.99,
			quantity:    -1,
			discount:    10,
			weight:      10.5,
			expectError: true,
			description: "Quantity cannot be negative",
		},
		{
			name:        "quantity above maximum",
			price:       99.99,
			quantity:    1001,
			discount:    10,
			weight:      10.5,
			expectError: true,
			description: "Quantity must not exceed 1000",
		},
		{
			name:        "discount not multiple of 5",
			price:       99.99,
			quantity:    10,
			discount:    7,
			weight:      10.5,
			expectError: true,
			description: "Discount must be multiple of 5",
		},
		{
			name:        "discount valid multiple",
			price:       99.99,
			quantity:    10,
			discount:    15,
			weight:      10.5,
			expectError: false,
			description: "Valid discount as multiple of 5",
		},
		{
			name:        "weight not multiple of 0.5",
			price:       99.99,
			quantity:    10,
			discount:    10,
			weight:      10.3,
			expectError: true,
			description: "Weight must be multiple of 0.5",
		},
		{
			name:        "weight valid multiple",
			price:       99.99,
			quantity:    10,
			discount:    10,
			weight:      10.5,
			expectError: false,
			description: "Valid weight as multiple of 0.5",
		},
		{
			name:        "all values valid",
			price:       99.99,
			quantity:    10,
			discount:    10,
			weight:      10.5,
			expectError: false,
			description: "All numeric validations pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"name":         "Valid Product Name",
				"sku":          "ABC-1234",
				"price":        tt.price,
				"quantity":     tt.quantity,
				"discount":     tt.discount,
				"status":       "pending",
				"category":     "electronics",
				"tags":         []string{"tag1", "tag2"},
				"seller_email": "seller@example.com",
				"product_id":   "123e4567-e89b-12d3-a456-426614174000",
				"created_at":   time.Now().Format(time.RFC3339),
				"weight":       tt.weight,
				"colors":       []string{"red"},
			}

			jsonBody, _ := json.Marshal(payload)
			body := bytes.NewBuffer(jsonBody)
			ctx, _ := NewTestContext(http.MethodPost, "/test", body)
			ctx.Request().Header.Set("Content-Type", "application/json")

			var product TestProduct
			err := ctx.Bind(&product)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestBind_EnumValidation(t *testing.T) {
	tests := []struct {
		name        string
		status      string
		category    string
		currency    string
		expectError bool
		description string
	}{
		{
			name:        "invalid status value",
			status:      "invalid",
			category:    "electronics",
			currency:    "USD",
			expectError: true,
			description: "Status must be one of: pending, paid, canceled, refunded",
		},
		{
			name:        "valid status pending",
			status:      "pending",
			category:    "electronics",
			currency:    "USD",
			expectError: false,
			description: "Valid status value",
		},
		{
			name:        "invalid category value",
			status:      "pending",
			category:    "invalid",
			currency:    "USD",
			expectError: true,
			description: "Category must be one of: electronics, clothing, books, food",
		},
		{
			name:        "valid category clothing",
			status:      "paid",
			category:    "clothing",
			currency:    "EUR",
			expectError: false,
			description: "Valid category value",
		},
		{
			name:        "invalid currency value",
			status:      "pending",
			category:    "electronics",
			currency:    "INVALID",
			expectError: true,
			description: "Currency must be one of: USD, EUR, GBP, CDF",
		},
		{
			name:        "valid currency CDF",
			status:      "canceled",
			category:    "books",
			currency:    "CDF",
			expectError: false,
			description: "Valid currency value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"name":         "Valid Product Name",
				"sku":          "ABC-1234",
				"price":        99.99,
				"status":       tt.status,
				"category":     tt.category,
				"currency":     tt.currency,
				"tags":         []string{"tag1", "tag2"},
				"seller_email": "seller@example.com",
				"product_id":   "123e4567-e89b-12d3-a456-426614174000",
				"created_at":   time.Now().Format(time.RFC3339),
				"weight":       10.5,
				"colors":       []string{"red"},
			}

			jsonBody, _ := json.Marshal(payload)
			body := bytes.NewBuffer(jsonBody)
			ctx, _ := NewTestContext(http.MethodPost, "/test", body)
			ctx.Request().Header.Set("Content-Type", "application/json")

			var product TestProduct
			err := ctx.Bind(&product)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestBind_SliceValidation(t *testing.T) {
	tests := []struct {
		name        string
		tags        []string
		images      []string
		dimensions  []float64
		colors      []string
		expectError bool
		description string
	}{
		{
			name:        "tags too few items",
			tags:        []string{"tag1"},
			images:      []string{"img1"},
			dimensions:  []float64{1, 2, 3},
			colors:      []string{"red"},
			expectError: true,
			description: "Tags must have at least 2 items",
		},
		{
			name:        "tags too many items",
			tags:        []string{"t1", "t2", "t3", "t4", "t5", "t6"},
			images:      []string{"img1"},
			dimensions:  []float64{1, 2, 3},
			colors:      []string{"red"},
			expectError: true,
			description: "Tags must not exceed 5 items",
		},
		{
			name:        "tags not unique",
			tags:        []string{"tag1", "tag1", "tag2"},
			images:      []string{"img1"},
			dimensions:  []float64{1, 2, 3},
			colors:      []string{"red"},
			expectError: true,
			description: "Tags must have unique items",
		},
		{
			name:        "images too many",
			tags:        []string{"t1", "t2"},
			images:      []string{"i1", "i2", "i3", "i4", "i5", "i6", "i7", "i8", "i9", "i10", "i11"},
			dimensions:  []float64{1, 2, 3},
			colors:      []string{"red"},
			expectError: true,
			description: "Images must not exceed 10 items",
		},
		{
			name:        "dimensions wrong count - too few",
			tags:        []string{"t1", "t2"},
			images:      []string{"img1"},
			dimensions:  []float64{1, 2},
			colors:      []string{"red"},
			expectError: true,
			description: "Dimensions must have exactly 3 items",
		},
		{
			name:        "dimensions wrong count - too many",
			tags:        []string{"t1", "t2"},
			images:      []string{"img1"},
			dimensions:  []float64{1, 2, 3, 4},
			colors:      []string{"red"},
			expectError: true,
			description: "Dimensions must have exactly 3 items",
		},
		{
			name:        "colors too few",
			tags:        []string{"t1", "t2"},
			images:      []string{"img1"},
			dimensions:  []float64{1, 2, 3},
			colors:      []string{},
			expectError: true,
			description: "Colors must have at least 1 item",
		},
		{
			name:        "colors not unique",
			tags:        []string{"t1", "t2"},
			images:      []string{"img1"},
			dimensions:  []float64{1, 2, 3},
			colors:      []string{"red", "red"},
			expectError: true,
			description: "Colors must have unique items",
		},
		{
			name:        "all slices valid",
			tags:        []string{"t1", "t2", "t3"},
			images:      []string{"img1", "img2"},
			dimensions:  []float64{1, 2, 3},
			colors:      []string{"red", "blue"},
			expectError: false,
			description: "All slice validations pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"name":         "Valid Product Name",
				"sku":          "ABC-1234",
				"price":        99.99,
				"status":       "pending",
				"category":     "electronics",
				"tags":         tt.tags,
				"images":       tt.images,
				"dimensions":   tt.dimensions,
				"colors":       tt.colors,
				"seller_email": "seller@example.com",
				"product_id":   "123e4567-e89b-12d3-a456-426614174000",
				"created_at":   time.Now().Format(time.RFC3339),
				"weight":       10.5,
			}

			jsonBody, _ := json.Marshal(payload)
			body := bytes.NewBuffer(jsonBody)
			ctx, _ := NewTestContext(http.MethodPost, "/test", body)
			ctx.Request().Header.Set("Content-Type", "application/json")

			var product TestProduct
			err := ctx.Bind(&product)

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestBind_FormatValidation(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		value       string
		expectError bool
		description string
	}{
		{
			name:        "invalid email format",
			field:       "seller_email",
			value:       "invalid-email",
			expectError: true,
			description: "Email must be valid format",
		},
		{
			name:        "valid email format",
			field:       "seller_email",
			value:       "valid@example.com",
			expectError: false,
			description: "Valid email address",
		},
		{
			name:        "invalid uuid format",
			field:       "product_id",
			value:       "not-a-uuid",
			expectError: true,
			description: "Product ID must be valid UUID",
		},
		{
			name:        "valid uuid format",
			field:       "product_id",
			value:       "123e4567-e89b-12d3-a456-426614174000",
			expectError: false,
			description: "Valid UUID format",
		},
		{
			name:        "invalid date format",
			field:       "launch_date",
			value:       "2024-13-45",
			expectError: true,
			description: "Date must be valid YYYY-MM-DD",
		},
		{
			name:        "valid date format",
			field:       "launch_date",
			value:       "2024-12-08",
			expectError: false,
			description: "Valid date format",
		},
		{
			name:        "invalid duration format",
			field:       "shipping_time",
			value:       "invalid",
			expectError: true,
			description: "Duration must be valid Go duration",
		},
		{
			name:        "valid duration format",
			field:       "shipping_time",
			value:       "24h30m",
			expectError: false,
			description: "Valid duration format",
		},
		{
			name:        "invalid uri format",
			field:       "seller_website",
			value:       "not a uri",
			expectError: true,
			description: "URI must be valid format",
		},
		{
			name:        "valid uri format",
			field:       "seller_website",
			value:       "https://example.com",
			expectError: false,
			description: "Valid URI format",
		},
		{
			name:        "invalid ipv4 format",
			field:       "seller_ip",
			value:       "999.999.999.999",
			expectError: true,
			description: "IPv4 must be valid format",
		},
		{
			name:        "valid ipv4 format",
			field:       "seller_ip",
			value:       "192.168.1.1",
			expectError: false,
			description: "Valid IPv4 format",
		},
		{
			name:        "invalid ipv6 format",
			field:       "server_ipv6",
			value:       "invalid",
			expectError: true,
			description: "IPv6 must be valid format",
		},
		{
			name:        "valid ipv6 format",
			field:       "server_ipv6",
			value:       "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expectError: false,
			description: "Valid IPv6 format",
		},
		{
			name:        "invalid hostname format",
			field:       "hostname",
			value:       "invalid_hostname!",
			expectError: true,
			description: "Hostname must be valid format",
		},
		{
			name:        "valid hostname format",
			field:       "hostname",
			value:       "example.com",
			expectError: false,
			description: "Valid hostname format",
		},
		{
			name:        "invalid phone regex",
			field:       "phone_number",
			value:       "123",
			expectError: true,
			description: "Phone number must match pattern",
		},
		{
			name:        "valid phone regex",
			field:       "phone_number",
			value:       "+243999999999",
			expectError: false,
			description: "Valid phone number format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"name":           "Valid Product Name",
				"sku":            "ABC-1234",
				"price":          99.99,
				"status":         "pending",
				"category":       "electronics",
				"tags":           []string{"tag1", "tag2"},
				"seller_email":   "seller@example.com",
				"product_id":     "123e4567-e89b-12d3-a456-426614174000",
				"created_at":     time.Now().Format(time.RFC3339),
				"weight":         10.5,
				"colors":         []string{"red"},
				"launch_date":    "2024-12-08",
				"shipping_time":  "24h",
				"seller_website": "https://example.com",
				"seller_ip":      "192.168.1.1",
				"server_ipv6":    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
				"hostname":       "example.com",
				"phone_number":   "+243999999999",
			}
			payload[tt.field] = tt.value

			jsonBody, _ := json.Marshal(payload)
			body := bytes.NewBuffer(jsonBody)
			ctx, _ := NewTestContext(http.MethodPost, "/test", body)
			ctx.Request().Header.Set("Content-Type", "application/json")

			var product TestProduct
			err := ctx.Bind(&product)

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestBind_DefaultValues(t *testing.T) {
	payload := map[string]interface{}{
		"name":         "Valid Product Name",
		"sku":          "ABC-1234",
		"price":        99.99,
		"status":       "pending",
		"category":     "electronics",
		"tags":         []string{"tag1", "tag2"},
		"seller_email": "seller@example.com",
		"product_id":   "123e4567-e89b-12d3-a456-426614174000",
		"created_at":   time.Now().Format(time.RFC3339),
		"weight":       10.5,
		"colors":       []string{"red"},
	}

	jsonBody, _ := json.Marshal(payload)
	body := bytes.NewBuffer(jsonBody)
	ctx, _ := NewTestContext(http.MethodPost, "/test", body)
	ctx.Request().Header.Set("Content-Type", "application/json")

	var product TestProduct
	if err := ctx.Bind(&product); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check default values
	if product.Quantity != 1 {
		t.Errorf("Expected default quantity 1, got %d", product.Quantity)
	}
	if product.ShippingTime != "24h" {
		t.Errorf("Expected default shipping_time '24h', got '%s'", product.ShippingTime)
	}
	if !product.IsActive {
		t.Errorf("Expected default is_active true, got false")
	}
	if product.Currency != "USD" {
		t.Errorf("Expected default currency 'USD', got '%s'", product.Currency)
	}
	if product.Priority != 0 {
		t.Errorf("Expected default priority 0, got %d", product.Priority)
	}
}

func TestBind_InvalidJSON(t *testing.T) {
	body := bytes.NewBufferString(`{invalid json}`)
	ctx, _ := NewTestContext(http.MethodPost, "/test", body)
	ctx.Request().Header.Set("Content-Type", "application/json")

	var product TestProduct
	err := ctx.Bind(&product)

	if err == nil {
		t.Error("Expected error for invalid JSON, got none")
	}
}

func TestBind_EmptyBody(t *testing.T) {
	body := bytes.NewBufferString(`{}`)
	ctx, _ := NewTestContext(http.MethodPost, "/test", body)
	ctx.Request().Header.Set("Content-Type", "application/json")

	var product TestProduct
	err := ctx.Bind(&product)

	if err == nil {
		t.Error("Expected error for empty body (missing required fields), got none")
	}
}
