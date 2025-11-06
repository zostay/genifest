package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// OutputMode represents the different output modes available.
type OutputMode string

const (
	ModeColor    OutputMode = "color"
	ModePlain    OutputMode = "plain"
	ModeMarkdown OutputMode = "markdown"
)

// Writer provides an interface for different output modes.
type Writer interface {
	// Printf formats and writes output with optional formatting
	Printf(format string, args ...interface{})
	// Header writes a header with emphasis
	Header(text string)
	// Success writes a success message
	Success(text string)
	// Warning writes a warning message
	Warning(text string)
	// Error writes an error message
	Error(text string)
	// Info writes an informational message
	Info(text string)
	// Bullet writes a bullet point with optional count/value
	Bullet(text string, value interface{})
	// Change writes a change description
	Change(file, path, oldVal, newVal string, docIndex int)
	// NoChange writes a no-change description
	NoChange(file, path, value string, docIndex int)
	// UpdatedFile writes an updated file message
	UpdatedFile(file string, changeCount int)
	// Println writes a line break
	Println()
}

// ColorWriter implements Writer for colored terminal output.
type ColorWriter struct {
	out io.Writer
}

// NewColorWriter creates a new ColorWriter.
func NewColorWriter(out io.Writer) *ColorWriter {
	return &ColorWriter{out: out}
}

func (w *ColorWriter) Printf(format string, args ...interface{}) {
	fmt.Fprintf(w.out, format, args...)
}

func (w *ColorWriter) Header(text string) {
	fmt.Fprintf(w.out, "🔍 \033[1;34m%s\033[0m\n", text)
}

func (w *ColorWriter) Success(text string) {
	fmt.Fprintf(w.out, "✅ \033[1;32m%s\033[0m\n", text)
}

func (w *ColorWriter) Warning(text string) {
	fmt.Fprintf(w.out, "⚠️  \033[33mWarning:\033[0m %s\n", text)
}

func (w *ColorWriter) Error(text string) {
	fmt.Fprintf(w.out, "❌ \033[1;31m%s\033[0m\n", text)
}

func (w *ColorWriter) Info(text string) {
	fmt.Fprintf(w.out, "\033[1m%s\033[0m\n", text)
}

func (w *ColorWriter) Bullet(text string, value interface{}) {
	fmt.Fprintf(w.out, "  \033[36m•\033[0m %s \033[36m%v\033[0m\n", text, value)
}

func (w *ColorWriter) Change(file, path, oldVal, newVal string, docIndex int) {
	fmt.Fprintf(w.out, "  ✏️  %s -> document[\033[35m%d\033[0m] -> \033[36m%s\033[0m: \033[31m%s\033[0m → \033[32m%s\033[0m\n",
		file, docIndex, path, oldVal, newVal)
}

func (w *ColorWriter) NoChange(file, path, value string, docIndex int) {
	fmt.Fprintf(w.out, "  ✓  %s -> document[\033[35m%d\033[0m] -> \033[36m%s\033[0m: \033[37m%s\033[0m (no change)\n",
		file, docIndex, path, value)
}

func (w *ColorWriter) UpdatedFile(file string, changeCount int) {
	fmt.Fprintf(w.out, "📝 \033[1;32mUpdated file:\033[0m %s (\033[36m%d\033[0m changes)\n", file, changeCount)
}

func (w *ColorWriter) Println() {
	fmt.Fprintln(w.out)
}

// PlainWriter implements Writer for plain text output without colors or emojis.
type PlainWriter struct {
	out io.Writer
}

// NewPlainWriter creates a new PlainWriter.
func NewPlainWriter(out io.Writer) *PlainWriter {
	return &PlainWriter{out: out}
}

func (w *PlainWriter) Printf(format string, args ...interface{}) {
	// Strip ANSI color codes from format string
	cleanFormat := stripANSI(format)
	fmt.Fprintf(w.out, cleanFormat, args...)
}

func (w *PlainWriter) Header(text string) {
	fmt.Fprintf(w.out, "%s\n", text)
}

func (w *PlainWriter) Success(text string) {
	fmt.Fprintf(w.out, "SUCCESS: %s\n", text)
}

func (w *PlainWriter) Warning(text string) {
	fmt.Fprintf(w.out, "WARNING: %s\n", text)
}

func (w *PlainWriter) Error(text string) {
	fmt.Fprintf(w.out, "ERROR: %s\n", text)
}

func (w *PlainWriter) Info(text string) {
	fmt.Fprintf(w.out, "%s\n", text)
}

func (w *PlainWriter) Bullet(text string, value interface{}) {
	fmt.Fprintf(w.out, "  • %s %v\n", text, value)
}

func (w *PlainWriter) Change(file, path, oldVal, newVal string, docIndex int) {
	fmt.Fprintf(w.out, "  CHANGED: %s -> document[%d] -> %s: %s → %s\n",
		file, docIndex, path, oldVal, newVal)
}

func (w *PlainWriter) NoChange(file, path, value string, docIndex int) {
	fmt.Fprintf(w.out, "  UNCHANGED: %s -> document[%d] -> %s: %s (no change)\n",
		file, docIndex, path, value)
}

func (w *PlainWriter) UpdatedFile(file string, changeCount int) {
	fmt.Fprintf(w.out, "Updated file: %s (%d changes)\n", file, changeCount)
}

func (w *PlainWriter) Println() {
	fmt.Fprintln(w.out)
}

// MarkdownWriter implements Writer for markdown output.
type MarkdownWriter struct {
	out io.Writer
}

// NewMarkdownWriter creates a new MarkdownWriter.
func NewMarkdownWriter(out io.Writer) *MarkdownWriter {
	return &MarkdownWriter{out: out}
}

func (w *MarkdownWriter) Printf(format string, args ...interface{}) {
	// Strip ANSI color codes and convert basic formatting
	cleanFormat := stripANSI(format)
	fmt.Fprintf(w.out, cleanFormat, args...)
}

func (w *MarkdownWriter) Header(text string) {
	fmt.Fprintf(w.out, "## %s\n\n", text)
}

func (w *MarkdownWriter) Success(text string) {
	fmt.Fprintf(w.out, "✅ **%s**\n\n", text)
}

func (w *MarkdownWriter) Warning(text string) {
	fmt.Fprintf(w.out, "⚠️  **Warning:** %s\n\n", text)
}

func (w *MarkdownWriter) Error(text string) {
	fmt.Fprintf(w.out, "❌ **ERROR:** %s\n\n", text)
}

func (w *MarkdownWriter) Info(text string) {
	fmt.Fprintf(w.out, "**%s**\n\n", text)
}

func (w *MarkdownWriter) Bullet(text string, value interface{}) {
	fmt.Fprintf(w.out, "- %s **%v**\n", text, value)
}

func (w *MarkdownWriter) Change(file, path, oldVal, newVal string, docIndex int) {
	fmt.Fprintf(w.out, "  - ✏️  `%s` → document[%d] → `%s`: `%s` → `%s`\n",
		file, docIndex, path, oldVal, newVal)
}

func (w *MarkdownWriter) NoChange(file, path, value string, docIndex int) {
	fmt.Fprintf(w.out, "  - ✓ `%s` → document[%d] → `%s`: `%s` (no change)\n",
		file, docIndex, path, value)
}

func (w *MarkdownWriter) UpdatedFile(file string, changeCount int) {
	fmt.Fprintf(w.out, "📝 **Updated file:** `%s` (%d changes)\n\n", file, changeCount)
}

func (w *MarkdownWriter) Println() {
	fmt.Fprintln(w.out)
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(str string) string {
	// Simple regex-like replacement for common ANSI codes
	// Replace escape sequences like \033[0m, \033[1;32m, etc.
	result := str

	// Common ANSI escape patterns
	patterns := []string{
		"\033[0m", "\033[1m", "\033[31m", "\033[32m", "\033[33m", "\033[34m",
		"\033[35m", "\033[36m", "\033[37m", "\033[1;31m", "\033[1;32m",
		"\033[1;33m", "\033[1;34m", "\033[1;35m", "\033[1;36m", "\033[1;37m",
	}

	for _, pattern := range patterns {
		result = strings.ReplaceAll(result, pattern, "")
	}

	return result
}

// NewWriter creates a new Writer based on the specified mode.
func NewWriter(mode OutputMode, out io.Writer) Writer {
	switch mode {
	case ModeColor:
		return NewColorWriter(out)
	case ModePlain:
		return NewPlainWriter(out)
	case ModeMarkdown:
		return NewMarkdownWriter(out)
	default:
		return NewPlainWriter(out)
	}
}

// DetectDefaultMode detects the appropriate default output mode based on TTY.
func DetectDefaultMode() OutputMode {
	// Check if stdout is a terminal
	if isTerminal(os.Stdout) {
		return ModeColor
	}
	return ModePlain
}

// isTerminal checks if the given file is a terminal.
func isTerminal(f *os.File) bool {
	// Check if file descriptor refers to a terminal
	// This is a simplified check - in production you might want to use
	// a library like golang.org/x/term for more robust detection
	stat, err := f.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (typical for terminals)
	return (stat.Mode() & os.ModeCharDevice) != 0
}