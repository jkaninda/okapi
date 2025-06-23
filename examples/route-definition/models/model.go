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
	Id    int    `json:"id"`
	Name  string `json:"name" form:"name"  max:"50" required:"true" description:"Book name"`
	Price int    `json:"price" form:"price" query:"price" yaml:"price" required:"true" description:"Book price"`
}
type ErrorResponse struct {
	Success bool `json:"success"`
	Status  int  `json:"status"`
	Details any  `json:"details"`
}

type AuthRequest struct {
	Username string `json:"username" required:"true" description:"Username for authentication"`
	Password string `json:"password" required:"true" description:"Password for authentication"`
}
type AuthResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Token     string `json:"token,omitempty"`
	ExpiresAt int64  `json:"expires,omitempty"`
}
type UserInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}
