package services

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorContext provides additional context for errors
type ErrorContext struct {
	Operation string
	TeamCode  string
	GameID    int
	File      string
	Line      int
	Function  string
}

// WrappedError contains an error with additional context
type WrappedError struct {
	Err     error
	Context ErrorContext
	Stack   []string
}

// Error implements the error interface
func (we *WrappedError) Error() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] %s", we.Context.Operation, we.Err.Error()))

	if we.Context.TeamCode != "" {
		sb.WriteString(fmt.Sprintf(" (team: %s)", we.Context.TeamCode))
	}

	if we.Context.GameID != 0 {
		sb.WriteString(fmt.Sprintf(" (game: %d)", we.Context.GameID))
	}

	if we.Context.File != "" {
		sb.WriteString(fmt.Sprintf(" at %s:%d", we.Context.File, we.Context.Line))
	}

	return sb.String()
}

// Unwrap returns the underlying error
func (we *WrappedError) Unwrap() error {
	return we.Err
}

// WrapError wraps an error with context information
func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Get caller information
	pc, file, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	funcName := ""
	if fn != nil {
		funcName = fn.Name()
	}

	// Extract just the filename (not full path)
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}

	return &WrappedError{
		Err: err,
		Context: ErrorContext{
			Operation: operation,
			File:      file,
			Line:      line,
			Function:  funcName,
		},
	}
}

// WrapErrorWithTeam wraps an error with team context
func WrapErrorWithTeam(err error, operation, teamCode string) error {
	if err == nil {
		return nil
	}

	wrapped := WrapError(err, operation)
	if we, ok := wrapped.(*WrappedError); ok {
		we.Context.TeamCode = teamCode
	}

	return wrapped
}

// WrapErrorWithGame wraps an error with game context
func WrapErrorWithGame(err error, operation string, gameID int) error {
	if err == nil {
		return nil
	}

	wrapped := WrapError(err, operation)
	if we, ok := wrapped.(*WrappedError); ok {
		we.Context.GameID = gameID
	}

	return wrapped
}

// WrapErrorWithContext wraps an error with full context
func WrapErrorWithContext(err error, ctx ErrorContext) error {
	if err == nil {
		return nil
	}

	// Get caller information
	pc, file, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	if fn != nil {
		ctx.Function = fn.Name()
	}

	// Extract just the filename
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		ctx.File = parts[len(parts)-1]
	}
	ctx.Line = line

	return &WrappedError{
		Err:     err,
		Context: ctx,
	}
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, target error) bool {
	if err == nil || target == nil {
		return false
	}

	// Check if it's a wrapped error
	if we, ok := err.(*WrappedError); ok {
		return IsErrorType(we.Err, target)
	}

	return err.Error() == target.Error()
}

// GetErrorContext extracts the context from a wrapped error
func GetErrorContext(err error) *ErrorContext {
	if we, ok := err.(*WrappedError); ok {
		return &we.Context
	}
	return nil
}

// FormatErrorChain formats the complete error chain for logging
func FormatErrorChain(err error) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder
	depth := 0

	for err != nil {
		indent := strings.Repeat("  ", depth)

		if we, ok := err.(*WrappedError); ok {
			sb.WriteString(fmt.Sprintf("%s[%d] %s\n", indent, depth, we.Error()))
			err = we.Err
		} else {
			sb.WriteString(fmt.Sprintf("%s[%d] %s\n", indent, depth, err.Error()))
			break
		}

		depth++
	}

	return sb.String()
}

// Common error types
var (
	ErrAPICallFailed     = fmt.Errorf("API call failed")
	ErrInvalidTeamCode   = fmt.Errorf("invalid team code")
	ErrGameNotFound      = fmt.Errorf("game not found")
	ErrDataNotAvailable  = fmt.Errorf("data not available")
	ErrCacheMiss         = fmt.Errorf("cache miss")
	ErrPersistenceFailed = fmt.Errorf("persistence failed")
	ErrModelNotTrained   = fmt.Errorf("model not trained")
	ErrInvalidInput      = fmt.Errorf("invalid input")
	ErrServiceNotInit    = fmt.Errorf("service not initialized")
	ErrRateLimitExceeded = fmt.Errorf("rate limit exceeded")
)

// SafeExecute executes a function and wraps any panic as an error
func SafeExecute(operation string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = WrapError(fmt.Errorf("panic: %v", r), operation)
		}
	}()

	return fn()
}

// SafeExecuteWithReturn executes a function with return value and wraps any panic
func SafeExecuteWithReturn[T any](operation string, fn func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			result = zero
			err = WrapError(fmt.Errorf("panic: %v", r), operation)
		}
	}()

	return fn()
}
