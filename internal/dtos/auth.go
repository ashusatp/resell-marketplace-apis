package dtos

import (
	"net/mail"
	"strings"
)

type SignUpRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (req *SignUpRequest) Validate() error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if len(req.Name) > 100 {
		return &ValidationError{Field: "name", Message: "name must not exceed 100 characters"}
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return &ValidationError{Field: "email", Message: "email is invalid"}
	}
	if len(req.Password) < 6 {
		return &ValidationError{Field: "password", Message: "password must be at least 6 characters long"}
	}
	if len(req.Password) > 72 {
		return &ValidationError{Field: "password", Message: "password must not exceed 72 characters"}
	}
	return nil
}

type SignUpResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (req *LoginRequest) Validate() error {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return &ValidationError{Field: "email", Message: "email is invalid"}
	}
	return nil
}

type LoginResponse struct {
	Token string `json:"token"`
}
