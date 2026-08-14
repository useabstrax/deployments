package abstrax

// ProjectResponse is the versioned project inspect API response.
type ProjectResponse struct {
	APIVersion string  `json:"api_version"`
	Project    Project `json:"project"`
}

// Project describes a project returned by abstrax project inspect --json.
type Project struct {
	Name     string           `json:"name"`
	Path     string           `json:"path"`
	User     string           `json:"user"`
	Runtime  ProjectRuntime   `json:"runtime"`
	Domains  []string         `json:"domains"`
	Services []ProjectService `json:"services"`
}

// ProjectRuntime describes a project's runtime.
type ProjectRuntime struct {
	Type    string `json:"type"`
	Version string `json:"version"`
}

// ProjectService describes a project-owned service.
type ProjectService struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Result is the standard Abstrax CLI JSON envelope for service commands.
type Result struct {
	Status    string `json:"status"`
	Action    string `json:"action"`
	Summary   string `json:"summary,omitempty"`
	Message   string `json:"message,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
}
