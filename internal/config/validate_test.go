package config

import (
	"strings"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Field:   "test.field",
		Message: "is invalid",
	}

	got := err.Error()
	want := "test.field: is invalid"

	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name   string
		errors ValidationErrors
		want   string
	}{
		{
			name:   "empty errors",
			errors: ValidationErrors{},
			want:   "",
		},
		{
			name: "single error",
			errors: ValidationErrors{
				{Field: "field1", Message: "is required"},
			},
			want: "field1: is required",
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				{Field: "field1", Message: "is required"},
				{Field: "field2", Message: "is invalid"},
			},
			want: "validation errors:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errors.Error()
			if !strings.Contains(got, tt.want) {
				t.Errorf("Error() = %q, want to contain %q", got, tt.want)
			}
		})
	}
}

func TestValidator_AddError(t *testing.T) {
	v := NewValidator()

	if v.HasErrors() {
		t.Error("new validator should not have errors")
	}

	v.AddError("field1", "error1")

	if !v.HasErrors() {
		t.Error("validator should have errors after adding one")
	}

	if len(v.Errors()) != 1 {
		t.Errorf("len(Errors()) = %d, want 1", len(v.Errors()))
	}
}

func TestValidator_Error(t *testing.T) {
	v := NewValidator()

	if err := v.Error(); err != nil {
		t.Errorf("Error() = %v, want nil for validator without errors", err)
	}

	v.AddError("field1", "error1")

	if err := v.Error(); err == nil {
		t.Error("Error() = nil, want error for validator with errors")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				App: AppConfig{
					Name:  "test-app",
					Debug: false,
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			config: &Config{
				App: AppConfig{
					Name:  "",
					Debug: false,
				},
			},
			wantErr: true,
		},
		{
			name: "name too short",
			config: &Config{
				App: AppConfig{
					Name:  "a",
					Debug: false,
				},
			},
			wantErr: true,
		},
		{
			name: "name too long",
			config: &Config{
				App: AppConfig{
					Name:  strings.Repeat("a", 51),
					Debug: false,
				},
			},
			wantErr: true,
		},
		{
			name: "name with whitespace",
			config: &Config{
				App: AppConfig{
					Name:  "test app",
					Debug: false,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
