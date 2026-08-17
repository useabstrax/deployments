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
