package search

import (
	"strings"
	"testing"
)

func TestQueryCandidatesNormalizesCompoundIdentifiers(t *testing.T) {
	candidates := QueryCandidates("LegacyServiceCoordinator")
	if len(candidates) == 0 || candidates[0] != "LegacyServiceCoordinator" {
		t.Fatalf("unexpected candidates: %v", candidates)
	}
	if !containsCandidate(candidates, "ServiceCoordinator") {
		t.Fatalf("expected suffix alias in candidates: %v", candidates)
	}
}

func TestQueryCandidatesExtractsTermsFromQuestion(t *testing.T) {
	candidates := QueryCandidates("what metadata service coordinator calls metadata provider for")
	if !containsCandidate(candidates, "ServiceCoordinator") || !containsCandidate(candidates, "MetadataProvider") {
		t.Fatalf("expected compound terms in candidates: %v", candidates)
	}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), "what") {
			t.Fatalf("stop word leaked into candidate %q", candidate)
		}
	}
}

func TestQueryCandidatesRejectsBlankQuery(t *testing.T) {
	if candidates := QueryCandidates("   "); len(candidates) != 0 {
		t.Fatalf("expected no candidates, got %v", candidates)
	}
}

func containsCandidate(candidates []string, wanted string) bool {
	for _, candidate := range candidates {
		if candidate == wanted {
			return true
		}
	}
	return false
}
