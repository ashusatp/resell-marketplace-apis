package dtos

import (
	"strings"
)

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
