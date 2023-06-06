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
	"testing"
)

func TestEntryStatusString(t *testing.T) {
	testCases := []struct {
		status EntryStatus
		want   string
	}{
		{EntryOk, "EntryOk"},
		{EntryInvalid, "EntryInvalid"},
		{EntryRequiresValidation, "EntryRequiresValidation"},
		{EntryLookupError, "EntryLookupError"},
		{EntryStatus(10), "Unknown state: 10"},
	}

	for _, tc := range testCases {
		got := tc.status.String()
		if got != tc.want {
			t.Errorf("Expected status.String() to return %q, but got %q", tc.want, got)
		}
	}
}
