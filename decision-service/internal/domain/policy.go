package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

type Rule struct {
	Effect    string `json:"effect"`
	ID        string `json:"id"`
	Operation string `json:"operation"`
	Priority  int    `json:"priority"`
	Region    string `json:"region"`
}

type Manifest struct {
	Revision  int    `json:"revision"`
	PageCount int    `json:"page_count"`
	RuleCount int    `json:"rule_count"`
	Digest    string `json:"digest"`
}

type BundlePage struct {
	Revision  int    `json:"revision"`
	PageIndex int    `json:"page_index"`
	Rules     []Rule `json:"rules"`
}

type DecisionRequest struct {
	SubjectID string `json:"subject_id"`
	Operation string `json:"operation"`
	Region    string `json:"region"`
}

type DecisionResponse struct {
	Decision       string `json:"decision"`
	MatchedRuleID  string `json:"matched_rule_id,omitempty"`
	PolicyRevision int    `json:"policy_revision"`
}

type Snapshot struct {
	revision int
	rules    []Rule
}

func NewSnapshot(revision int, rules []Rule) (*Snapshot, error) {
	if revision < 0 {
		return nil, errors.New("revision cannot be negative")
	}
	ordered := append([]Rule(nil), rules...)
	seen := make(map[string]struct{}, len(ordered))
	for _, rule := range ordered {
		if rule.ID == "" {
			return nil, errors.New("rule id cannot be empty")
		}
		if _, exists := seen[rule.ID]; exists {
			return nil, fmt.Errorf("duplicate rule id %q", rule.ID)
		}
		seen[rule.ID] = struct{}{}
		if rule.Effect != "allow" && rule.Effect != "deny" {
			return nil, fmt.Errorf("rule %q has invalid effect", rule.ID)
		}
		if rule.Operation == "" || rule.Region == "" {
			return nil, fmt.Errorf("rule %q has an empty matcher", rule.ID)
		}
	}
	sortRules(ordered)
	return &Snapshot{revision: revision, rules: ordered}, nil
}

func (s *Snapshot) Revision() int {
	return s.revision
}

func (s *Snapshot) Evaluate(request DecisionRequest) DecisionResponse {
	response := DecisionResponse{Decision: "deny", PolicyRevision: s.revision}
	for _, rule := range s.rules {
		if matches(rule.Operation, request.Operation) && matches(rule.Region, request.Region) {
			response.Decision = rule.Effect
			response.MatchedRuleID = rule.ID
			return response
		}
	}
	return response
}

func DigestRules(rules []Rule) (string, error) {
	canonical := append([]Rule(nil), rules...)
	sortRules(canonical)
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func sortRules(rules []Rule) {
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority == rules[j].Priority {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].Priority < rules[j].Priority
	})
}

func matches(expected, actual string) bool {
	return expected == "*" || expected == actual
}
