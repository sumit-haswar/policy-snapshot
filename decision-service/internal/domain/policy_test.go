package domain

import "testing"

func TestSnapshotEvaluatesOrderedRules(t *testing.T) {
	snapshot, err := NewSnapshot(7, []Rule{
		{Effect: "allow", ID: "allow-export", Operation: "export", Priority: 20, Region: "*"},
		{Effect: "deny", ID: "deny-restricted", Operation: "*", Priority: 10, Region: "restricted"},
	})
	if err != nil {
		t.Fatal(err)
	}

	response := snapshot.Evaluate(DecisionRequest{SubjectID: "subject-1", Operation: "export", Region: "restricted"})
	if response.Decision != "deny" || response.MatchedRuleID != "deny-restricted" || response.PolicyRevision != 7 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestSnapshotRejectsDuplicateRules(t *testing.T) {
	_, err := NewSnapshot(1, []Rule{
		{Effect: "allow", ID: "same", Operation: "read", Priority: 1, Region: "*"},
		{Effect: "deny", ID: "same", Operation: "write", Priority: 2, Region: "*"},
	})
	if err == nil {
		t.Fatal("expected duplicate rule error")
	}
}

func TestDigestIsIndependentOfInputOrder(t *testing.T) {
	a := Rule{Effect: "allow", ID: "a", Operation: "read", Priority: 20, Region: "*"}
	b := Rule{Effect: "deny", ID: "b", Operation: "export", Priority: 10, Region: "restricted"}
	first, err := DigestRules([]Rule{a, b})
	if err != nil {
		t.Fatal(err)
	}
	second, err := DigestRules([]Rule{b, a})
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("digests differ: %s != %s", first, second)
	}
}

func TestDigestMatchesPublishedCanonicalExample(t *testing.T) {
	rules := []Rule{
		{Effect: "deny", ID: "deny-restricted-region", Operation: "*", Priority: 10, Region: "restricted"},
		{Effect: "deny", ID: "deny-standard-export", Operation: "export", Priority: 15, Region: "standard"},
		{Effect: "allow", ID: "allow-read", Operation: "read", Priority: 20, Region: "*"},
		{Effect: "allow", ID: "allow-standard-export", Operation: "export", Priority: 30, Region: "standard"},
	}
	digest, err := DigestRules(rules)
	if err != nil {
		t.Fatal(err)
	}
	const expected = "sha256:00812c5276238b2410a813dc0de75d6c4b829d1ee1542332c1491a913ef61ecc"
	if digest != expected {
		t.Fatalf("digest = %q, want %q", digest, expected)
	}
}
