package abstrax_test

import (
	"encoding/json"
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func TestProjectResponseUnmarshal(t *testing.T) {
	raw := `{
		"api_version": "v1",
		"project": {
			"name": "example",
			"path": "/var/www/example.com",
			"user": "example",
			"runtime": {"type": "php", "version": "8.5"},
			"domains": ["example.com"],
			"services": [{"name": "example-worker", "type": "worker"}]
		}
	}`

	var resp abstrax.ProjectResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.APIVersion != "v1" {
		t.Fatalf("api_version = %q", resp.APIVersion)
	}
	if resp.Project.Name != "example" {
		t.Fatalf("project name = %q", resp.Project.Name)
	}
	if len(resp.Project.Services) != 1 {
		t.Fatalf("services = %d, want 1", len(resp.Project.Services))
	}
}

func TestResultUnmarshal(t *testing.T) {
	raw := `{"status":"success","action":"project.service.restart","summary":"ok"}`
	var result abstrax.Result
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "success" {
		t.Fatalf("status = %q", result.Status)
	}
}
