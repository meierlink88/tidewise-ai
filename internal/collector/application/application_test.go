package application

import (
	"context"
	"errors"
	"testing"
)

func TestRetryStateWriteIsBounded(t *testing.T) {
	attempts := 0
	err := retryStateWrite(func(context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary database failure")
		}
		return nil
	})
	if err != nil || attempts != 3 {
		t.Fatalf("eventual success err=%v attempts=%d", err, attempts)
	}

	attempts = 0
	err = retryStateWrite(func(context.Context) error {
		attempts++
		return errors.New("persistent database failure")
	})
	if err == nil || attempts != 3 {
		t.Fatalf("bounded failure err=%v attempts=%d", err, attempts)
	}
}
