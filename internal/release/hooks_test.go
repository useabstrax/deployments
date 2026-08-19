package release

import "testing"

func TestCombineHookOutputIncludesStdout(t *testing.T) {
	got := combineHookOutput("artisan error here", "composer progress")
	if got != "composer progress\nartisan error here" {
		t.Fatalf("got %q", got)
	}
}

func TestTailLines(t *testing.T) {
	in := "a\nb\nc\nd\ne"
	got := tailLines(in, 3)
	if got != "c\nd\ne" {
		t.Fatalf("got %q", got)
	}
}

func TestHookInvokesAbstrax(t *testing.T) {
	cases := map[string]bool{
		`abstrax composer run --project="$ABSTRAX_PROJECT" install`: true,
		`/usr/bin/abstrax composer run install`:                     true,
		`abstrax-composer run install`:                              true,
		`FOO=1 abstrax composer run install`:                        true,
		`$ABSTRAX_CLI_PHP artisan migrate --force`:                  false,
		`composer install --no-dev`:                                 false,
		`npm ci && npm run build`:                                   false,
	}
	for hook, want := range cases {
		if got := hookInvokesAbstrax(hook); got != want {
			t.Fatalf("%q: got %v want %v", hook, got, want)
		}
	}
}
