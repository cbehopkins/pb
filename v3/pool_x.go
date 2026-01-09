//go:build linux || darwin || freebsd || netbsd || openbsd || solaris || dragonfly || plan9 || aix
// +build linux darwin freebsd netbsd openbsd solaris dragonfly plan9 aix

package pb

import (
	"fmt"
	"os"
	"strings"

	"github.com/cbehopkins/pb/v3/termutil"
)

func (p *Pool) print(first bool) bool {
	p.m.Lock()
	defer p.m.Unlock()
	var out string
	if !first {
		out = fmt.Sprintf("\033[%dA", p.lastBarsCount)
	}
	isFinished := true
	bars := p.bars
	rows, cols, err := termutil.TerminalSize()
	if err != nil {
		cols = defaultBarWidth
	}
	if rows > 0 && len(bars) > rows {
		// we need to hide bars that overflow terminal height
		bars = bars[len(bars)-rows:]
	}
	for _, bar := range bars {
		if !bar.IsFinished() {
			isFinished = false
		}
		bar.SetWidth(cols)
		result := bar.String()
		if r := cols - CellCount(result); r > 0 {
			result += strings.Repeat(" ", r)
		}
		out += fmt.Sprintf("\r%s\n", result)
	}
	var printErr error
	if p.Output != nil {
		_, printErr = fmt.Fprint(p.Output, out)
	} else {
		_, printErr = fmt.Fprint(os.Stderr, out)
	}
	if printErr != nil {
		// Log write errors to stderr as a fallback
		fmt.Fprintf(os.Stderr, "pool print error: %v\n", printErr)
	}
	p.lastBarsCount = len(bars)
	return isFinished
}
