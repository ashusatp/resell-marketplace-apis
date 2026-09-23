package handlers

import (
	"fmt"
	"net/mail"
	"strings"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (ve *ValidationError) Error() string {
	if ve.Field == "" {
		return ve.Message
	}
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

type CreateListingRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Price       int64  `json:"price" binding:"required"`
	City        string `json:"city" binding:"required"`
}

func (req *CreateListingRequest) Validate() error {

	if strings.TrimSpace(req.Title) == "" {
		return &ValidationError{Field: "title", Message: "title is required"}
	}
	if strings.TrimSpace(req.Description) == "" {
		return &ValidationError{Field: "description", Message: "description is required"}
	}
	if req.Price <= 0 {
		return &ValidationError{Field: "price", Message: "price must be greater than 0"}
	}
	if strings.TrimSpace(req.City) == "" {
		return &ValidationError{Field: "city", Message: "city is required"}
	}
	return nil

}

type CreateListingResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Created string `json:"created_at"`
}

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
