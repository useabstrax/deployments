package gitx_test

import (
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/gitx"
)

func TestClassifyRef(t *testing.T) {
	cases := []struct {
		in   string
		kind gitx.RefKind
	}{
		{"main", gitx.RefBranch},
		{"abcdef0", gitx.RefSHA},
		{"0123456789abcdef0123456789abcdef01234567", gitx.RefSHA},
		{"tags/v1.0.0", gitx.RefTag},
		{"refs/tags/v1.0.0", gitx.RefTag},
	}
	for _, tc := range cases {
		if got := gitx.ClassifyRef(tc.in); got != tc.kind {
			t.Fatalf("%q: got %v want %v", tc.in, got, tc.kind)
		}
	}
}

func TestNormalizeTag(t *testing.T) {
	if got := gitx.NormalizeTag("refs/tags/v1"); got != "v1" {
		t.Fatalf("got %q", got)
	}
}
