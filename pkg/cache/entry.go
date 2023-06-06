package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
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
	// Body is the entry body as a serialized http.Response
	Body []byte

	// Timestamp is the time the body was last modified.
	Timestamp int64
}

// TODO: Benchmark, encoding/decoding might slow down the hot path.

// Encode encodes an entry into a byte array.
func (e *Entry) Encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(e); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeEntry decodes a byte array into an Entry.
func DecodeEntry(data []byte) (*Entry, error) {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var entry *Entry
	if err := dec.Decode(&entry); err != nil {
		return &Entry{}, err
	}
	return entry, nil
}
