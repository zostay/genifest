package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ValidationError represents a validation error with a path to the problematic field.
type ValidationError struct {
	Path     string
	Message  string
	Value    interface{}
	Filename string // The filename where the error occurred
}

// Error implements the error interface.
func (ve *ValidationError) Error() string {
	var prefix string
	if ve.Filename != "" {
		prefix = fmt.Sprintf("❌ %s: ", ve.Filename)
	} else {
		prefix = "❌ "
	}
	
	if ve.Value != nil {
		return fmt.Sprintf("%s%s is %q which %s", prefix, ve.Path, ve.Value, ve.Message)
	}
	return fmt.Sprintf("%s%s %s", prefix, ve.Path, ve.Message)
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

// ErrorWithContext creates a validation error with the current path, message, and context filename.
func (pb *PathBuilder) ErrorWithContext(message string, ctx *ValidationContext) *ValidationError {
	ve := NewValidationError(pb.String(), message)
	if ctx != nil {
		ve.Filename = ctx.Filename
	}
	return ve
}

// ErrorWithValue creates a validation error with the current path, message, and value.
func (pb *PathBuilder) ErrorWithValue(message string, value interface{}) *ValidationError {
	return NewValidationErrorWithValue(pb.String(), message, value)
}

// ErrorWithValueAndContext creates a validation error with the current path, message, value, and context filename.
func (pb *PathBuilder) ErrorWithValueAndContext(message string, value interface{}, ctx *ValidationContext) *ValidationError {
	ve := NewValidationErrorWithValue(pb.String(), message, value)
	if ctx != nil {
		ve.Filename = ctx.Filename
	}
	return ve
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
// the current path for function scope resolution, path building, and filename information.
type ValidationContext struct {
	CloudHome   string
	Functions   []FunctionDefinition
	CurrentPath string
	PathBuilder *PathBuilder
	Filename    string // The filename where the configuration being validated was loaded from
}

// WithPath creates a new validation context with the given path builder.
func (ctx *ValidationContext) WithPath(pb *PathBuilder) *ValidationContext {
	return &ValidationContext{
		CloudHome:   ctx.CloudHome,
		Functions:   ctx.Functions,
		CurrentPath: ctx.CurrentPath,
		PathBuilder: pb,
		Filename:    ctx.Filename,
	}
}

// WithField creates a new validation context with the given field added to the path.
func (ctx *ValidationContext) WithField(name string) *ValidationContext {
	var pb *PathBuilder
	if ctx.PathBuilder != nil {
		pb = ctx.PathBuilder.Field(name)
	} else {
		pb = NewPathBuilder("").Field(name)
	}
	return ctx.WithPath(pb)
}

// WithIndex creates a new validation context with the given index added to the path.
func (ctx *ValidationContext) WithIndex(index int) *ValidationContext {
	var pb *PathBuilder
	if ctx.PathBuilder != nil {
		pb = ctx.PathBuilder.Index(index)
	} else {
		pb = NewPathBuilder("").Index(index)
	}
	return ctx.WithPath(pb)
}

// safeError returns a ValidationError using the context's path builder.
func safeError(ctx *ValidationContext, message string) error {
	if ctx.PathBuilder != nil {
		return ctx.PathBuilder.ErrorWithContext(message, ctx)
	}
	return fmt.Errorf("%s", message)
}

// safeErrorWithValue returns a ValidationError with value using the context's path builder.
func safeErrorWithValue(ctx *ValidationContext, fieldName string, message string, value interface{}) error {
	if ctx.PathBuilder != nil {
		return ctx.PathBuilder.ErrorWithValueAndContext(message, value, ctx)
	}
	return fmt.Errorf("%s '%v' %s", fieldName, value, message)
}

// safeErrorWithField returns a ValidationError using the context's path builder.
func safeErrorWithField(ctx *ValidationContext, fieldName string, message string) error {
	if ctx.PathBuilder != nil {
		return ctx.PathBuilder.ErrorWithContext(message, ctx)
	}
	return fmt.Errorf("%s validation failed: %s", fieldName, message)
}
