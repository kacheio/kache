package cache

import "time"

// Entry is the cache entry.
type Entry struct {
	Body         []byte
	LastModified time.Time
}
