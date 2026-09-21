package policy

import "testing"

func TestStoreReturnsInitialSnapshot(t *testing.T) {
	store := NewStore(BootstrapSnapshot())
	if got := store.Current().Revision(); got != 1 {
		t.Fatalf("revision = %d, want 1", got)
	}
}
