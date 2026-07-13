package doctor

import "testing"

func TestAnalyzeSummarizesChecks(t *testing.T) {
	report := Analyze(t.TempDir())
	if report.OK || report.Total == 0 || report.Failed == 0 {
		t.Fatalf("expected failed report for empty directory: %+v", report)
	}
	if report.Passed+report.Failed != report.Total || len(report.Checks) != report.Total {
		t.Fatalf("inconsistent report counters: %+v", report)
	}
}
