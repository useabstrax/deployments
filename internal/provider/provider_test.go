package provider_test

import (
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/provider"
)

func TestNormalizeRepository(t *testing.T) {
	p := provider.GitHub{}
	cases := map[string]string{
		"git@github.com:acme/app.git":     "git@github.com:acme/app.git",
		"https://github.com/acme/app.git": "git@github.com:acme/app.git",
		"acme/app":                        "git@github.com:acme/app.git",
	}
	for in, want := range cases {
		got, err := p.NormalizeRepository(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}
