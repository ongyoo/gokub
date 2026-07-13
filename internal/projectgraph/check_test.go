package projectgraph

import "testing"

func TestAnalyzeFindsBoundaryAndCycleViolations(t *testing.T) {
	graph := Graph{
		Module: "example.com/service",
		Nodes: []Node{
			{ID: "domain", Layer: "domain"},
			{ID: "platform", Layer: "platform"},
		},
		Edges: []Edge{{From: "domain", To: "platform"}, {From: "platform", To: "domain"}},
	}
	report := Analyze(graph)
	if report.OK || len(report.Violations) != 2 {
		t.Fatalf("expected boundary and cycle violations: %+v", report)
	}
	seen := map[string]bool{}
	for _, violation := range report.Violations {
		seen[violation.Type] = true
	}
	if !seen["boundary"] || !seen["cycle"] {
		t.Fatalf("missing violation types: %+v", report.Violations)
	}
}

func TestAnalyzeAcceptsInwardDependencies(t *testing.T) {
	graph := Graph{
		Nodes: []Node{{ID: "cmd", Layer: "service"}, {ID: "domain", Layer: "domain"}, {ID: "shared", Layer: "shared"}},
		Edges: []Edge{{From: "cmd", To: "domain"}, {From: "domain", To: "shared"}},
	}
	if report := Analyze(graph); !report.OK || len(report.Violations) != 0 {
		t.Fatalf("valid dependencies were rejected: %+v", report)
	}
}
