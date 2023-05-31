package cache

import (
	"fmt"
	"time"
)

// EntryStatus is the state of a cached response.
type EntryStatus int

const (
	// EntryInvalid indicates that the cached response is not usable or valid (cache miss).
	EntryInvalid EntryStatus = iota

	// EntryOk indicates that the cached response is valid and can be used (cache hit).
	EntryOk

	// EntryRequiresValidation indicates that the cached response needs to be validated.
	EntryRequiresValidation

	// EntryError indicates an error occurred while retrieving the response.
	EntryLookupError
)

// String returns the Entry Status as a string.
func (s EntryStatus) String() string {
	switch s {
	case EntryOk:
		return "EntryOk"
	case EntryInvalid:
		return "EntryInvalid"
	case EntryRequiresValidation:
		return "EntryRequiresValidation"
	case EntryLookupError:
		return "EntryLookupError"
	default:
		return fmt.Sprintf("Unknown state: %d", s)
	}
}

// Entry is the cache entry.
type Entry struct {
	Body         []byte
	LastModified time.Time
}
