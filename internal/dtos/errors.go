package dtos

import "fmt"

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
