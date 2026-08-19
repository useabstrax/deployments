package plugin

import (
	"encoding/json"
	"io"
	"strings"
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
	Action      string `json:"action,omitempty"`
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
			deployCommand("setup", "Guided one-shot deploy setup for a project"),
			deployCommand("init", "Scaffold deploy directories and deploy.json"),
			deployCommand("configure", "Show or update deploy configuration"),
			deployCommand("key", "Create or manage the project deploy key"),
			deployCommand("now", "Deploy a new release with zero downtime"),
			deployCommand("rollback", "Point current at a previous release"),
			deployCommand("list", "List releases for a project"),
			deployCommand("status", "Show deploy status and current release"),
			deployCommand("hooks", "List or edit deploy hooks"),
			deployCommand("version", "Display plugin version information"),
		},
	}
}

func deployCommand(name, description string) MetadataCommand {
	return MetadataCommand{
		Name:        name,
		Action:      "plugin." + PluginName + "." + strings.ReplaceAll(name, "-", "_"),
		Description: description,
	}
}

// WriteMetadata encodes metadata as indented JSON to w.
func WriteMetadata(w io.Writer, metadata Metadata) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(metadata)
}
