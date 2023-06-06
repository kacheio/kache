// MIT License
//
// Copyright (c) 2023 kache.io
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
