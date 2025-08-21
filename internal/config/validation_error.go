package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ValidationError represents a validation error with a path to the problematic field.
type ValidationError struct {
	Path    string
	Message string
	Value   interface{}
}

// Error implements the error interface.
func (ve *ValidationError) Error() string {
	if ve.Value != nil {
		return fmt.Sprintf("Validation Error: %s is %q which %s", ve.Path, ve.Value, ve.Message)
	}
	return fmt.Sprintf("Validation Error: %s %s", ve.Path, ve.Message)
}

// NewValidationError creates a new validation error with the given path and message.
func NewValidationError(path, message string) *ValidationError {
	return &ValidationError{
		Path:    path,
		Message: message,
	}
}

// NewValidationErrorWithValue creates a new validation error with the given path, message, and value.
func NewValidationErrorWithValue(path, message string, value interface{}) *ValidationError {
	return &ValidationError{
		Path:    path,
		Message: message,
		Value:   value,
	}
}

// PathBuilder helps build validation error paths incrementally.
type PathBuilder struct {
	segments []string
}

// NewPathBuilder creates a new path builder starting with an optional root path.
func NewPathBuilder(root string) *PathBuilder {
	pb := &PathBuilder{}
	if root != "" {
		pb.segments = []string{root}
	}
	return pb
}

// Field adds a field name to the path (e.g., ".metadata").
func (pb *PathBuilder) Field(name string) *PathBuilder {
	newPB := &PathBuilder{
		segments: make([]string, len(pb.segments)+1),
	}
	copy(newPB.segments, pb.segments)
	newPB.segments[len(pb.segments)] = "." + name
	return newPB
}

// Index adds an array index to the path (e.g., "[0]").
func (pb *PathBuilder) Index(index int) *PathBuilder {
	newPB := &PathBuilder{
		segments: make([]string, len(pb.segments)+1),
	}
	copy(newPB.segments, pb.segments)
	newPB.segments[len(pb.segments)] = "[" + strconv.Itoa(index) + "]"
	return newPB
}

// String returns the complete path as a string.
func (pb *PathBuilder) String() string {
	if len(pb.segments) == 0 {
		return "."
	}

	result := strings.Join(pb.segments, "")
	if !strings.HasPrefix(result, ".") {
		result = "." + result
	}
	return result
}

// Error creates a validation error with the current path and given message.
func (pb *PathBuilder) Error(message string) *ValidationError {
	return NewValidationError(pb.String(), message)
}

// ErrorWithValue creates a validation error with the current path, message, and value.
func (pb *PathBuilder) ErrorWithValue(message string, value interface{}) *ValidationError {
	return NewValidationErrorWithValue(pb.String(), message, value)
}

// WrapError wraps an existing error with the current path context.
// If the error is already a ValidationError, it prepends the current path.
func (pb *PathBuilder) WrapError(err error) error {
	if err == nil {
		return nil
	}

	var ve *ValidationError
	if errors.As(err, &ve) {
		// If it's already a ValidationError, prepend our path
		newPath := pb.String()
		if newPath != "." && !strings.HasPrefix(ve.Path, newPath) {
			ve.Path = newPath + ve.Path
		}
		return ve
	}

	// If it's a regular error, create a new ValidationError
	return &ValidationError{
		Path:    pb.String(),
		Message: err.Error(),
	}
}

// ValidationContext provides context for validation including available functions,
// the current path for function scope resolution, and path building.
type ValidationContext struct {
	CloudHome   string
	Functions   []FunctionDefinition
	CurrentPath string
	PathBuilder *PathBuilder
}

// WithPath creates a new validation context with the given path builder.
func (ctx *ValidationContext) WithPath(pb *PathBuilder) *ValidationContext {
	if ctx == nil {
		return &ValidationContext{PathBuilder: pb}
	}

	return &ValidationContext{
		CloudHome:   ctx.CloudHome,
		Functions:   ctx.Functions,
		CurrentPath: ctx.CurrentPath,
		PathBuilder: pb,
	}
}

// WithField creates a new validation context with the given field added to the path.
func (ctx *ValidationContext) WithField(name string) *ValidationContext {
	var pb *PathBuilder
	if ctx != nil && ctx.PathBuilder != nil {
		pb = ctx.PathBuilder.Field(name)
	} else {
		pb = NewPathBuilder("").Field(name)
	}
	return ctx.WithPath(pb)
}

// WithIndex creates a new validation context with the given index added to the path.
func (ctx *ValidationContext) WithIndex(index int) *ValidationContext {
	var pb *PathBuilder
	if ctx != nil && ctx.PathBuilder != nil {
		pb = ctx.PathBuilder.Index(index)
	} else {
		pb = NewPathBuilder("").Index(index)
	}
	return ctx.WithPath(pb)
}
