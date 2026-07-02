package fsrs

import (
	"errors"
	"testing"
)

func TestErrorType(t *testing.T) {
	t.Run("Error() returns message", func(t *testing.T) {
		err := &Error{Code: ErrCodeInvalidInput, Message: "test message"}
		if err.Error() != "test message" {
			t.Errorf("expected 'test message', got=%v", err.Error())
		}
	})

	t.Run("errors.Is matches by code", func(t *testing.T) {
		err := &Error{Code: ErrCodeManualRating, Message: "different instance"}
		if !errors.Is(err, ErrManualRating) {
			t.Error("expected errors.Is to match by code")
		}
	})

	t.Run("errors.Is does not match different code", func(t *testing.T) {
		err := &Error{Code: ErrCodeInvalidRating, Message: "different code"}
		if errors.Is(err, ErrManualRating) {
			t.Error("expected errors.Is NOT to match different code")
		}
	})

	t.Run("errors.Is works with singleton vars", func(t *testing.T) {
		if !errors.Is(ErrManualRating, ErrManualRating) {
			t.Error("expected singleton to match itself")
		}
		if !errors.Is(ErrInvalidRating, ErrInvalidRating) {
			t.Error("expected singleton to match itself")
		}
		if errors.Is(ErrManualRating, ErrInvalidRating) {
			t.Error("expected different singletons NOT to match")
		}
	})

	t.Run("errors.As extracts code", func(t *testing.T) {
		err := &Error{Code: ErrCodeManualStateRequired, Message: "state needed"}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatal("expected errors.As to extract *Error")
		}
		if fsrsErr.Code != ErrCodeManualStateRequired {
			t.Errorf("expected code %v, got=%v", ErrCodeManualStateRequired, fsrsErr.Code)
		}
	})

	t.Run("existing error messages unchanged", func(t *testing.T) {
		if ErrManualRating.Error() != "fsrs: cannot rollback a manual rating" {
			t.Errorf("ErrManualRating message changed: %v", ErrManualRating.Error())
		}
		if ErrInvalidRating.Error() != "fsrs: invalid rating for rollback" {
			t.Errorf("ErrInvalidRating message changed: %v", ErrInvalidRating.Error())
		}
		if ErrManualStateRequired.Error() != "fsrs: state is required for manual rating" {
			t.Errorf("ErrManualStateRequired message changed: %v", ErrManualStateRequired.Error())
		}
		if ErrManualDueRequired.Error() != "fsrs: due is required for manual rating when state is not New" {
			t.Errorf("ErrManualDueRequired message changed: %v", ErrManualDueRequired.Error())
		}
	})

	t.Run("errors.Is returns false for non-*Error target", func(t *testing.T) {
		if errors.Is(ErrManualRating, errors.New("fsrs: cannot rollback a manual rating")) {
			t.Error("expected errors.Is to return false for plain error")
		}
	})

	t.Run("Is handles nil safely", func(t *testing.T) {
		var nilErr *Error
		if errors.Is(ErrManualRating, nilErr) {
			t.Error("expected errors.Is with nil target to return false")
		}
		if nilErr.Is(ErrManualRating) {
			t.Error("expected nil receiver.Is to return false")
		}
		if nilErr.Error() != "fsrs: <nil>" {
			t.Errorf("expected nil Error() to return 'fsrs: <nil>', got=%v", nilErr.Error())
		}
	})

	t.Run("errors.As negative case", func(t *testing.T) {
		plainErr := errors.New("plain error")
		var fsrsErr *Error
		if errors.As(plainErr, &fsrsErr) {
			t.Error("expected errors.As to fail for non-*Error")
		}
	})

	t.Run("sentinel codes match constants", func(t *testing.T) {
		if ErrManualRating.Code != ErrCodeManualRating {
			t.Errorf("ErrManualRating.Code=%v, want ErrCodeManualRating=%v", ErrManualRating.Code, ErrCodeManualRating)
		}
		if ErrInvalidRating.Code != ErrCodeInvalidRating {
			t.Errorf("ErrInvalidRating.Code=%v, want ErrCodeInvalidRating=%v", ErrInvalidRating.Code, ErrCodeInvalidRating)
		}
		if ErrManualStateRequired.Code != ErrCodeManualStateRequired {
			t.Errorf("ErrManualStateRequired.Code=%v, want ErrCodeManualStateRequired=%v", ErrManualStateRequired.Code, ErrCodeManualStateRequired)
		}
		if ErrManualDueRequired.Code != ErrCodeManualDueRequired {
			t.Errorf("ErrManualDueRequired.Code=%v, want ErrCodeManualDueRequired=%v", ErrManualDueRequired.Code, ErrCodeManualDueRequired)
		}
	})
}
