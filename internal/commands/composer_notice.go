package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/presets"
)

const composerInstallHint = `sudo abstrax plugin install composer && sudo abstrax composer setup`

// maybeWarnComposerPlugin prints a notice when a PHP/Laravel-style preset
// is used and the Abstrax Composer plugin does not appear to be available.
func maybeWarnComposerPlugin(p *output.Printer, preset string) {
	name := strings.ToLower(strings.TrimSpace(preset))
	if name != string(presets.Laravel) && name != "php" {
		return
	}
	if composerPluginAvailable() {
		return
	}

	p.Warn("Laravel/PHP deploys use the Abstrax Composer plugin in after_clone hooks.")
	p.Warn("Install and set it up with:")
	if !globals.JSON && !globals.JSONStream {
		fmt.Fprintf(p.Out, "\n  %s\n\n", composerInstallHint)
	} else {
		p.Warn("%s", composerInstallHint)
	}
}

func composerPluginAvailable() bool {
	if _, err := exec.LookPath("abstrax-composer"); err == nil {
		return true
	}
	// Delegated via abstrax binary when the plugin is installed.
	cmd := exec.Command("abstrax", "composer", "version")
	if err := cmd.Run(); err == nil {
		return true
	}
	return false
}
