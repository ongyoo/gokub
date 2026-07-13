package gokub

import "testing"

func TestReleasedModuleVersion(t *testing.T) {
	for _, test := range []struct {
		module string
		want   string
	}{
		{module: "v0.2.3", want: "0.2.3"},
		{module: "v0.2.3-beta.1", want: "0.2.3-beta.1"},
		{module: "v0.2.3-0.20260713064030-ff757a12cbb1", want: "0.1.0"},
		{module: "v0.2.3-0.20260713064030-ff757a12cbb1+dirty", want: "0.1.0"},
		{module: "(devel)", want: "0.1.0"},
		{module: "", want: "0.1.0"},
	} {
		if got := releasedModuleVersion("0.1.0", test.module); got != test.want {
			t.Fatalf("releasedModuleVersion(%q) = %q, want %q", test.module, got, test.want)
		}
	}
}
