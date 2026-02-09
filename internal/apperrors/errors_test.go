package apperrors

import (
	"errors"
	"strings"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		contains []string
	}{
		{
			name: "with message",
			err: &AppError{
				Op:      "test.operation",
				Err:     errors.New("underlying error"),
				Message: "user friendly message",
			},
			contains: []string{"test.operation", "user friendly message", "underlying error"},
		},
		{
			name: "without message",
			err: &AppError{
				Op:  "test.operation",
				Err: errors.New("underlying error"),
			},
			contains: []string{"test.operation", "underlying error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("Error() = %q, want to contain %q", got, want)
				}
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &AppError{
		Op:  "test.operation",
		Err: underlying,
	}

	if unwrapped := err.Unwrap(); unwrapped != underlying {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlying)
	}
}

func TestAppError_ErrorsIs(t *testing.T) {
	tests := []struct {
		name   string
		err    *AppError
		target error
		want   bool
	}{
		{
			name: "matches ErrNotFound via Unwrap",
			err: &AppError{
				Op:  "test.operation",
				Err: ErrNotFound,
			},
			target: ErrNotFound,
			want:   true,
		},
		{
			name: "does not match",
			err: &AppError{
				Op:  "test.operation",
				Err: ErrNotFound,
			},
			target: ErrInvalidInput,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	err := New("test.op", errors.New("test error"), "test message")

	if err.Op != "test.op" {
		t.Errorf("Op = %q, want %q", err.Op, "test.op")
	}

	if err.Message != "test message" {
		t.Errorf("Message = %q, want %q", err.Message, "test message")
	}

	if err.Err == nil {
		t.Error("Err is nil")
	}
}

func TestWrap(t *testing.T) {
	underlying := errors.New("test error")
	err := Wrap("test.op", underlying)

	if err.Op != "test.op" {
		t.Errorf("Op = %q, want %q", err.Op, "test.op")
	}

	if err.Err != underlying {
		t.Errorf("Err = %v, want %v", err.Err, underlying)
	}
}

func TestWrapf(t *testing.T) {
	underlying := errors.New("test error")
	err := Wrapf("test.op", underlying, "context: %s", "value")

	if err.Op != "test.op" {
		t.Errorf("Op = %q, want %q", err.Op, "test.op")
	}

	if !strings.Contains(err.Message, "context: value") {
		t.Errorf("Message = %q, want to contain %q", err.Message, "context: value")
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "is not found",
			err:  ErrNotFound,
			want: true,
		},
		{
			name: "wrapped not found",
			err:  Wrap("test.op", ErrNotFound),
			want: true,
		},
		{
			name: "is not not found",
			err:  ErrInvalidInput,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInvalidInput(t *testing.T) {
	if !IsInvalidInput(ErrInvalidInput) {
		t.Error("IsInvalidInput(ErrInvalidInput) = false, want true")
	}

	if IsInvalidInput(ErrNotFound) {
		t.Error("IsInvalidInput(ErrNotFound) = true, want false")
	}
}

func TestIsUnauthorized(t *testing.T) {
	if !IsUnauthorized(ErrUnauthorized) {
		t.Error("IsUnauthorized(ErrUnauthorized) = false, want true")
	}

	if IsUnauthorized(ErrNotFound) {
		t.Error("IsUnauthorized(ErrNotFound) = true, want false")
	}
}

func TestIsInternal(t *testing.T) {
	if !IsInternal(ErrInternal) {
		t.Error("IsInternal(ErrInternal) = false, want true")
	}

	if IsInternal(ErrNotFound) {
		t.Error("IsInternal(ErrNotFound) = true, want false")
	}
}

func TestJoin(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	joined := Join(err1, err2)
	if joined == nil {
		t.Fatal("Join() returned nil")
	}

	if !errors.Is(joined, err1) {
		t.Error("joined error does not contain err1")
	}
	if !errors.Is(joined, err2) {
		t.Error("joined error does not contain err2")
	}
}
