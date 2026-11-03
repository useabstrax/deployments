package commands

import (
	"reflect"
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/release"
)

func TestParseSetJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "empty string", raw: "", want: []string{}},
		{name: "empty array", raw: "[]", want: []string{}},
		{name: "ordered list", raw: `["composer install","php artisan migrate"]`, want: []string{"composer install", "php artisan migrate"}},
		{name: "commas inside commands", raw: `["echo a,b","echo c"]`, want: []string{"echo a,b", "echo c"}},
		{name: "null json", raw: "null", want: []string{}},
		{name: "not an array", raw: `"just a string"`, wantErr: true},
		{name: "array of numbers", raw: `[1,2]`, wantErr: true},
		{name: "object", raw: `{"a":1}`, wantErr: true},
		{name: "invalid json", raw: `[`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseSetJSON(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNormalizeSetValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []string
		want   []string
	}{
		{name: "nil", values: nil, want: []string{}},
		{name: "empty slice", values: []string{}, want: []string{}},
		{name: "single empty clears", values: []string{""}, want: []string{}},
		{name: "single whitespace clears", values: []string{"  "}, want: []string{}},
		{name: "one command", values: []string{"composer install"}, want: []string{"composer install"}},
		{name: "multiple preserve order", values: []string{"a", "b", "c"}, want: []string{"a", "b", "c"}},
		{name: "commas preserved", values: []string{"echo a,b"}, want: []string{"echo a,b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeSetValues(tt.values)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMutuallyExclusiveHookMutators(t *testing.T) {
	t.Parallel()

	if err := mutuallyExclusiveHookMutators(true, false, false, false); err != nil {
		t.Fatalf("single mutator: %v", err)
	}
	if err := mutuallyExclusiveHookMutators(true, true, false, false); err == nil {
		t.Fatal("expected error for set+set-json")
	}
	if err := mutuallyExclusiveHookMutators(false, false, true, true); err == nil {
		t.Fatal("expected error for append+clear")
	}
}

func TestSetHooksPhases(t *testing.T) {
	t.Parallel()

	cfg := config.Default("example.com")
	hooks := []string{"one", "two"}
	if err := setHooks(&cfg, release.PhaseAfterClone, hooks); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.Hooks.AfterClone, hooks) {
		t.Fatalf("after_clone = %#v", cfg.Hooks.AfterClone)
	}
	if err := setHooks(&cfg, "nope", hooks); err == nil {
		t.Fatal("expected unknown phase error")
	}
}
