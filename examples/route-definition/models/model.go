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

package models

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    Book   `json:"data"`
}
type Book struct {
	Id       int    `json:"id"`
	Name     string `json:"name" form:"name"  max:"50" required:"true" description:"Book name"`
	Price    int    `json:"price" form:"price" query:"price" yaml:"price" required:"true" description:"Book price"`
	Year     int    `json:"year" form:"year" query:"year" yaml:"year" deprecated:"true" description:"Publication year"`
	Quantity int    `json:"quantity" form:"quantity" query:"quantity" yaml:"quantity" hidden:"true" description:"Available quantity"`
}
type ErrorResponseDto struct {
	Success bool `json:"success"`
	Status  int  `json:"status"`
	Details any  `json:"details"`
}

type AuthRequest struct {
	Username string `json:"username" required:"true" description:"Username for authentication"`
	Password string `json:"password" required:"true" description:"Password for authentication"`
}
type AuthResponse struct {
	Token     string   `json:"token,omitempty"`
	ExpiresAt int64    `json:"expires,omitempty"`
	User      UserInfo `json:"user,omitempty"`
}
type UserInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Example of Okapi using Body Field Style

type BookUpdateRequest struct {
	ID   int `path:"id"`
	Body struct {
		Name  string `json:"name"  minLength:"5" maxLength:"50" required:"true" description:"Book name"`
		Price int    `json:"price" min:"5" required:"true" description:"Book price"`
	} `json:"body"`
}
type BookRequest struct {
	Authorization string `header:"Authorization"`
	Body          Book   `json:"body"`
}
type BookResponse struct {
	RequestId string `header:"X-Request-Id"`
	Status    int    `json:"status"`
	Body      Book   `json:"body"`
}
type ResponseBase struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type ResponseDto[T any] struct {
	*ResponseBase
	Data T `json:"data,omitempty"`
}
type BooksResponse = ResponseDto[[]Book]

func SuccessResponse[T any](message string, data T) ResponseDto[T] {
	return ResponseDto[T]{
		ResponseBase: &ResponseBase{
			Success: true,
			Message: message,
		},
		Data: data,
	}
}
func ErrorResponse(message string, err error) ResponseDto[any] {
	return ResponseDto[any]{
		ResponseBase: &ResponseBase{
			Success: false,
			Message: message,
			Details: err.Error(),
		},
	}
}
