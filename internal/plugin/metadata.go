package plugin

import (
	"encoding/json"
	"io"
)

const (
	// ProtocolVersion is the supported plugin metadata protocol version.
	ProtocolVersion = 1

	PluginName      = "deploy"
	DisplayName     = "Deploy Plugin"
	Description     = "Zero-downtime GitHub deployments for Abstrax projects"
	Homepage        = "https://plugins.useabstrax.com/plugins/deploy"
	RequiresAbstrax = ">=0.1.0"
)

// MetadataCommand describes a subcommand exposed by the plugin.
type MetadataCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Metadata is the plugin metadata protocol v1 response.
type Metadata struct {
	ProtocolVersion int               `json:"protocol_version"`
	Name            string            `json:"name"`
	DisplayName     string            `json:"display_name"`
	Description     string            `json:"description"`
	Version         string            `json:"version"`
	RequiresAbstrax string            `json:"requires_abstrax"`
	Homepage        string            `json:"homepage,omitempty"`
	Commands        []MetadataCommand `json:"commands"`
}

// DefaultMetadata returns the plugin metadata for this build.
func DefaultMetadata() Metadata {
	return Metadata{
		ProtocolVersion: ProtocolVersion,
		Name:            PluginName,
		DisplayName:     DisplayName,
		Description:     Description,
		Version:         Version,
		RequiresAbstrax: RequiresAbstrax,
		Homepage:        Homepage,
		Commands: []MetadataCommand{
			{Name: "setup", Description: "Guided one-shot deploy setup for a project"},
			{Name: "init", Description: "Scaffold deploy directories and deploy.json"},
			{Name: "configure", Description: "Show or update deploy configuration"},
			{Name: "key", Description: "Create or manage the project deploy key"},
			{Name: "now", Description: "Deploy a new release with zero downtime"},
			{Name: "rollback", Description: "Point current at a previous release"},
			{Name: "list", Description: "List releases for a project"},
			{Name: "status", Description: "Show deploy status and current release"},
			{Name: "hooks", Description: "List or edit deploy hooks"},
			{Name: "version", Description: "Display plugin version information"},
		},
	}
}

// WriteMetadata encodes metadata as indented JSON to w.
func WriteMetadata(w io.Writer, metadata Metadata) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(metadata)
}
