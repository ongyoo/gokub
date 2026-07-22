package goversion

import "testing"

func TestPolicy(t *testing.T) {
	for version, want := range map[string]Support{
		"1.26": Latest, "1.25": Supported, "1.24": Unsupported, "1.27": Future,
	} {
		if got := Classify(version); got != want {
			t.Fatalf("Classify(%q) = %q, want %q", version, got, want)
		}
	}
	if err := Validate("go1.26.1"); err == nil {
		t.Fatal("patch or prefixed version was accepted")
	}
	if got := ParseGoMod("module example.com/api\n\ngo 1.25\n"); got != "1.25" {
		t.Fatalf("ParseGoMod() = %q", got)
	}
	if got := ParseGoMod("module example.com/api\n\ngo 1.25.7\n"); got != "1.25" {
		t.Fatalf("ParseGoMod(patch) = %q, want 1.25", got)
	}
}
