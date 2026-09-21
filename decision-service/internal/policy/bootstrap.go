package policy

import "example.com/policy-snapshot/decision-service/internal/domain"

func BootstrapSnapshot() *domain.Snapshot {
	snapshot, err := domain.NewSnapshot(1, []domain.Rule{
		{Effect: "deny", ID: "deny-restricted-region", Operation: "*", Priority: 10, Region: "restricted"},
		{Effect: "allow", ID: "allow-read", Operation: "read", Priority: 20, Region: "*"},
		{Effect: "allow", ID: "allow-standard-export", Operation: "export", Priority: 30, Region: "standard"},
	})
	if err != nil {
		panic(err)
	}
	return snapshot
}
