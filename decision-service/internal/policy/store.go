package policy

import "example.com/policy-snapshot/decision-service/internal/domain"

type Store struct {
	current *domain.Snapshot
}

func NewStore(initial *domain.Snapshot) *Store {
	if initial == nil {
		panic("initial snapshot cannot be nil")
	}
	return &Store{current: initial}
}

func (s *Store) Current() *domain.Snapshot {
	return s.current
}
