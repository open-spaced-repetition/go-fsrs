package fsrs

// ErrorCode identifies a specific error condition returned by FSRS operations.
type ErrorCode int

const (
	ErrCodeInvalidParameters ErrorCode = iota
	ErrCodeInvalidInput
	ErrCodeManualRating
	ErrCodeInvalidRating
	ErrCodeManualStateRequired
	ErrCodeManualDueRequired
)

// Error represents a structured FSRS error with a machine-readable code
// and a human-readable message.
type Error struct {
	Code    ErrorCode
	Message string
}

// Error returns the human-readable error message. Implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return "fsrs: <nil>"
	}
	return e.Message
}

// Is reports whether this error matches the target by comparing ErrorCode values.
// Returns true when both are nil. Returns false when only one is nil or the
// target is not an *Error.
func (e *Error) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	t, ok := target.(*Error)
	if !ok || t == nil {
		return false
	}
	return e.Code == t.Code
}

var (
	// ErrManualRating is returned by Rollback when the review log entry represents
	// a manual (non-graded) operation that cannot be rolled back.
	ErrManualRating = &Error{
		Code:    ErrCodeManualRating,
		Message: "fsrs: cannot rollback a manual rating",
	}

	// ErrInvalidRating is returned by Rollback when the review log rating is
	// outside the valid range [Again, Easy].
	ErrInvalidRating = &Error{
		Code:    ErrCodeInvalidRating,
		Message: "fsrs: invalid rating for rollback",
	}

	// ErrManualStateRequired is returned by Reschedule when a manual review entry
	// is missing the required State field.
	ErrManualStateRequired = &Error{
		Code:    ErrCodeManualStateRequired,
		Message: "fsrs: state is required for manual rating",
	}

	// ErrManualDueRequired is returned by Reschedule when a manual review entry
	// with a non-New state is missing the required Due field.
	ErrManualDueRequired = &Error{
		Code:    ErrCodeManualDueRequired,
		Message: "fsrs: due is required for manual rating when state is not New",
	}
)
