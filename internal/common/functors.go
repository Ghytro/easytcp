package common

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// WithTimeout launches a function that must finish in given timeout. If it doesn't manage to finish
// in given time boundaries, the cleanup callback is called to fix all the goroutine leaks possible.
// Cleanup is also called when function finishes with an error. Cleanup can return error to give additional
// info about the errors occured while cleanup. Guaranteed that cleanup callback will be called once
func WithTimeout[T any](initiator T, timeout time.Duration, fn func(T) error, cleanup func(T) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return WithContext(ctx, initiator, fn, cleanup)
}

// WithContext launches a function that must finish before the given context expires. If it doesn't manage to finish
// in given time boundaries, the cleanup callback is called to fix all the goroutine leaks possible.
// Cleanup is also called when function finishes with an error. Cleanup can return error to give additional
// info about the errors occured while cleanup. Guaranteed that cleanup callback will be called once
func WithContext[T any](ctx context.Context, initiator T, fn func(T) error, cleanup func(T) error) error {
	ready := make(chan struct{})
	once := sync.Once{}
	var timeOutErr *WithTimeoutError
	go func() {
		if err := fn(initiator); err != nil {
			once.Do(func() {
				timeOutErr = &WithTimeoutError{
					Err:        err,
					CleanupErr: cleanup(initiator),
				}
			})
		}
		ready <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		once.Do(func() {
			timeOutErr = &WithTimeoutError{
				Err:        errors.New("execution timed out"),
				CleanupErr: cleanup(initiator),
			}
		})
	case <-ready:
		break
	}
	if timeOutErr != nil {
		return timeOutErr
	}
	return nil
}

type WithTimeoutError struct {
	Err, CleanupErr error
}

func (e WithTimeoutError) Error() string {
	if e.CleanupErr == nil {
		return e.Err.Error()
	}
	return e.Unwrap().Error()
}

func (e WithTimeoutError) Unwrap() error {
	if e.CleanupErr == nil {
		return e.Err
	}
	return fmt.Errorf("%v (%w)", e.Err, e.CleanupErr)
}
