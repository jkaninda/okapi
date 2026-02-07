---
title: Validations
layout: default
parent: Features
nav_order: 8
---
# Validations
Okapi provides a powerful validation system that allows you to easily validate incoming request data against defined rules and constraints. This helps ensure that your API receives well-formed and expected data, improving the robustness and reliability of your application.

## Validation and Default Values

Okapi provides declarative validation and automatic default value assignment using struct tags.

### Basic Validation Tags

| Field Type | Tag                            | Description                                              |
|------------|--------------------------------|----------------------------------------------------------|
| `string`   | `minLength:"10"`               | Ensures the string has at least 10 characters.           |
| `string`   | `maxLength:"50"`               | Ensures the string does not exceed 50 characters.        |
| `number`   | `min:"5"`                      | Ensures the number is greater than or equal to 5.        |
| `number`   | `max:"100"`                    | Ensures the number is less than or equal to 100.         |
| `number`   | `multipleOf:"5"`               | Ensures the number is a multiple of the given value.     |
| `slice`    | `maxItems:"5"`                 | Ensures the slice contains at most 5 items.              |
| `slice`    | `minItems:"2"`                 | Ensures the slice contains at least 2 items.             |
| `slice`    | `uniqueItems:"true"`           | Ensures all items in the slice are unique.               |
| `any`      | `required:"true"`              | Marks the field as required.                             |
| `any`      | `default:"..."`                | Assigns a default value when the field is missing/empty. |
| `any`      | `enum:"pending,paid,canceled"` | Restricts the field to one of the listed values.         |
| `any`      | `format:"email"`               | Enables format validation (e.g. `email`, `uuid`, etc.).  |
| `any`      | `pattern:"^[a-zA-Z]+$"`        | Validates the field against a regular expression.        |



### Data Type & Format Validation

| Field Type  | Tag / Attribute                               | Description                                            |
|-------------|-----------------------------------------------|--------------------------------------------------------|
| `date`      | `format:"date"`                               | Validates the field as a date (YYYY-MM-DD).            |
| `date-time` | `format:"date-time"`                          | Validates the field as a date and time (RFC3339).      |
| `email`     | `format:"email"`                              | Validates the field as a valid email address.          |
| `duration`  | `format:"duration"`                           | Validates the field as a Go duration (e.g., `1h30m`).  |
| `uuid`      | `format:"uuid"`                               | Validates the field as a valid UUID.                   |
| `hostname`  | `format:"hostname"`                           | Validates the field as a valid hostname.               |
| `ipv4`      | `format:"ipv4"`                               | Validates the field as a valid IPv4 address.           |
| `ipv6`      | `format:"ipv6"`                               | Validates the field as a valid IPv6 address.           |
| `uri`       | `format:"uri"`                                | Validates the field as a valid URI.                    |
| `regex`     | `format:"regex" pattern="^\+?[1-9]\d{1,14}$"` | Validates the field using a custom regular expression. |

### Example

```go
type CreateUserRequest struct {
Email    string   `json:"email" required:"true" format:"email" example:"user@example.com"`
Password string   `json:"password" minLength:"8" description:"User password"`
Age      int      `json:"age" min:"18" max:"120" default:"18"`
Roles    []string `json:"roles" minItems:"1" uniqueItems:"true"`
}
```
To validate incoming request data, simply call `c.Bind()` with your struct. Okapi will automatically apply the defined validation rules and return an error if any validation fails.

```go
o.Post("/users", func(c *okapi.Context) error {
    var req CreateUserRequest
    if err := c.Bind(&req); err != nil {
        return c.ErrorBadRequest(err)
    }
    // Proceed with creating the user using validated data in req
    return c.JSON(http.StatusOK, req)
})
```