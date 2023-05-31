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
