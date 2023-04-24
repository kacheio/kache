package cache

import "testing"

func TestNewKey(t *testing.T) {
	// t.Errorf("not implemented")
}

func TestHashKey(t *testing.T) {

	key := Key{}
	key.Host = "example.com"

	hash := StableHashKey(key)

	expected := uint64(2492773929771664174)

	if hash != expected {
		t.Errorf("Expected hash %d, got %d", expected, hash)
	}

}
