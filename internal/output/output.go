package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Result is the standard structured result matching Abstrax CLI conventions.
type Result struct {
	Status    string      `json:"status"`
	Action    string      `json:"action"`
	Summary   string      `json:"summary,omitempty"`
	Message   string      `json:"message,omitempty"`
	ErrorCode string      `json:"error_code,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// ProgressEvent is one NDJSON progress line under --json-stream.
type ProgressEvent struct {
	Type    string `json:"type"`
	Action  string `json:"action"`
	Step    string `json:"step"`
	Message string `json:"message"`
}

type streamResult struct {
	Type      string      `json:"type"`
	Status    string      `json:"status"`
	Action    string      `json:"action"`
	Summary   string      `json:"summary,omitempty"`
	Message   string      `json:"message,omitempty"`
	ErrorCode string      `json:"error_code,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// Success builds a successful result.
func Success(action, summary string, data interface{}) Result {
	return Result{Status: "success", Action: action, Summary: summary, Data: data}
}

// Failure builds a failure result.
func Failure(action, errorCode, message string) Result {
	return Result{Status: "error", Action: action, ErrorCode: errorCode, Message: message}
}

// ProgressEmitter streams deploy progress compatible with Abstrax core json-stream.
type ProgressEmitter interface {
	Progress(action, step, message string)
	Info(format string, args ...interface{})
	Verbose(format string, args ...interface{})
	DryRun(format string, args ...interface{})
	Warn(format string, args ...interface{})
}

// Printer writes human, --json, or --json-stream output.
type Printer struct {
	Out        io.Writer
	Err        io.Writer
	JSONMode   bool
	JSONStream bool
	Quiet      bool
	VerboseOn  bool
	NoColor    bool
}

// NewPrinter creates a Printer writing to stdout/stderr.
func NewPrinter(jsonMode, jsonStream, quiet, verbose, noColor bool) *Printer {
	return &Printer{
		Out:        os.Stdout,
		Err:        os.Stderr,
		JSONMode:   jsonMode,
		JSONStream: jsonStream,
		Quiet:      quiet,
		VerboseOn:  verbose,
		NoColor:    noColor,
	}
}

func (p *Printer) machine() bool {
	return p.JSONMode || p.JSONStream
}

// Progress emits one flushed NDJSON progress line when stream mode is on,
// otherwise prints a human step message on TTY.
func (p *Printer) Progress(action, step, message string) {
	if p.JSONStream {
		writeNDJSON(p.Out, ProgressEvent{
			Type:    "progress",
			Action:  action,
			Step:    step,
			Message: message,
		})
		return
	}
	if p.machine() || p.Quiet {
		return
	}
	fmt.Fprintf(p.Out, "==> %s\n", message)
}

// Info prints an informational message unless quiet/machine.
func (p *Printer) Info(format string, args ...interface{}) {
	if p.Quiet || p.machine() {
		return
	}
	fmt.Fprintf(p.Out, format+"\n", args...)
}

// Verbose prints only when --verbose and not machine mode.
func (p *Printer) Verbose(format string, args ...interface{}) {
	if !p.VerboseOn || p.machine() {
		return
	}
	fmt.Fprintf(p.Out, "[verbose] "+format+"\n", args...)
}

// DryRun prints a dry-run notice.
func (p *Printer) DryRun(format string, args ...interface{}) {
	if p.machine() {
		return
	}
	fmt.Fprintf(p.Out, "[dry-run] "+format+"\n", args...)
}

// Warn prints a warning to stderr.
func (p *Printer) Warn(format string, args ...interface{}) {
	if p.machine() {
		return
	}
	fmt.Fprintf(p.Err, "WARNING: "+format+"\n", args...)
}

// Print writes a final result for --json / --json-stream / human.
func (p *Printer) Print(r Result) {
	if p.JSONStream {
		writeNDJSON(p.Out, streamResult{
			Type:      "result",
			Status:    r.Status,
			Action:    r.Action,
			Summary:   r.Summary,
			Message:   r.Message,
			ErrorCode: r.ErrorCode,
			Data:      r.Data,
		})
		return
	}
	if p.JSONMode {
		enc := json.NewEncoder(p.Out)
		enc.SetIndent("", "  ")
		_ = enc.Encode(r)
		return
	}
	if r.Status == "error" {
		fmt.Fprintf(p.Err, "ERROR: %s\n", r.Message)
		return
	}
	if r.Summary != "" {
		fmt.Fprintln(p.Out, r.Summary)
	}
}

// Line prints a plain line (for tables/lists) unless machine mode.
func (p *Printer) Line(format string, args ...interface{}) {
	if p.machine() {
		return
	}
	fmt.Fprintf(p.Out, format+"\n", args...)
}

func writeNDJSON(w io.Writer, v interface{}) {
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
	if f, ok := w.(*os.File); ok {
		_ = f.Sync()
	}
}
