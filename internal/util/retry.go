package util

import (
	"time"
)

// TODO: Implement more sophisticated retry mechanisms if needed,
// e.g., with exponential backoff, jitter, max attempts.

// SimpleFixedRetry is an example.
func SimpleFixedRetry(attempts int, sleep time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if i < attempts-1 { // Don't sleep after the last attempt
			time.Sleep(sleep)
		}
	}
	return err
}
