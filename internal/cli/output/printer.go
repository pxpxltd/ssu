package output

import (
	"fmt"
	"io"
)

// Printer provides formatted output helpers with color and symbols.
// It respects whatever color state fatih/color is in (NO_COLOR, TTY detection
// happen at the fatih/color level, not here).
type Printer struct {
	w io.Writer
}

// NewPrinter creates a Printer that writes to w.
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w}
}

// Success prints a green checkmark followed by the message.
func (p *Printer) Success(msg string) {
	Success.Fprint(p.w, Check+" ")
	fmt.Fprintln(p.w, msg)
}

// Successf prints a green checkmark followed by a formatted message.
func (p *Printer) Successf(format string, args ...any) {
	p.Success(fmt.Sprintf(format, args...))
}

// Error prints a red cross followed by the message.
func (p *Printer) Error(msg string) {
	Error.Fprint(p.w, Cross+" ")
	fmt.Fprintln(p.w, msg)
}

// Errorf prints a red cross followed by a formatted message.
func (p *Printer) Errorf(format string, args ...any) {
	p.Error(fmt.Sprintf(format, args...))
}

// Warning prints a yellow arrow followed by the message.
func (p *Printer) Warning(msg string) {
	Warning.Fprint(p.w, Arrow+" ")
	fmt.Fprintln(p.w, msg)
}

// Warningf prints a yellow arrow followed by a formatted message.
func (p *Printer) Warningf(format string, args ...any) {
	p.Warning(fmt.Sprintf(format, args...))
}

// Info prints a cyan bullet followed by the message.
func (p *Printer) Info(msg string) {
	Info.Fprint(p.w, Bullet+" ")
	fmt.Fprintln(p.w, msg)
}

// Infof prints a cyan bullet followed by a formatted message.
func (p *Printer) Infof(format string, args ...any) {
	p.Info(fmt.Sprintf(format, args...))
}
