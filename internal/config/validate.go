package config

import (
	"fmt"
	"strings"

	"github.com/thirdlf03/spec-tdd/internal/apperrors"
)

// ValidationError represents a validation error with field information
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface for multiple validation errors
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}

	var sb strings.Builder
	sb.WriteString("validation errors:\n")
	for _, err := range e {
		sb.WriteString("  - ")
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}
	return sb.String()
}

// Validator provides validation functionality for configuration
type Validator struct {
	errors ValidationErrors
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// AddError adds a validation error
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// HasErrors returns true if there are any validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all validation errors
func (v *Validator) Errors() ValidationErrors {
	return v.errors
}

// Error returns validation errors as a single error
func (v *Validator) Error() error {
	if !v.HasErrors() {
		return nil
	}
	return v.errors
}

// Validate validates the configuration using a validator
func (c *Config) Validate() error {
	v := NewValidator()

	// Validate app name
	if c.App.Name == "" {
		v.AddError("app.name", "is required")
	} else if len(c.App.Name) < 2 {
		v.AddError("app.name", "must be at least 2 characters")
	} else if len(c.App.Name) > 50 {
		v.AddError("app.name", "must be at most 50 characters")
	}

	// Validate app name doesn't contain invalid characters
	if strings.ContainsAny(c.App.Name, " \t\n\r") {
		v.AddError("app.name", "must not contain whitespace")
	}

	// Add more validation rules here as needed
	// Example: validate other configuration fields
	// if c.App.Port < 1 || c.App.Port > 65535 {
	//     v.AddError("app.port", "must be between 1 and 65535")
	// }

	if v.HasErrors() {
		return apperrors.Wrap("config.Validate", v.Error())
	}

	return nil
}
