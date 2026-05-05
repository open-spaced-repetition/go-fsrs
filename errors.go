package fsrs

import "errors"

var (
	// ErrManualRating is returned by Rollback when the review log entry represents
	// a manual (non-graded) operation that cannot be rolled back.
	ErrManualRating = errors.New("fsrs: cannot rollback a manual rating")

	// ErrInvalidRating is returned by Rollback when the review log rating is
	// outside the valid range [Again, Easy].
	ErrInvalidRating = errors.New("fsrs: invalid rating for rollback")

	// ErrManualStateRequired is returned by Reschedule when a manual review entry
	// is missing the required State field.
	ErrManualStateRequired = errors.New("fsrs: state is required for manual rating")

	// ErrManualDueRequired is returned by Reschedule when a manual review entry
	// with a non-New state is missing the required Due field.
	ErrManualDueRequired = errors.New("fsrs: due is required for manual rating when state is not New")
)
