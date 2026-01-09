//go:build windows
// +build windows

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
		coords, err := termutil.GetCursorPos()
		if err != nil {
			// Graceful fallback if cursor positioning fails
			fmt.Fprintf(os.Stderr, "cursor position error: %v\n", err)
			// Continue without repositioning
		} else {
			coords.Y -= int16(p.lastBarsCount)
			if coords.Y < 0 {
				coords.Y = 0
			}
			coords.X = 0

			err = termutil.SetCursorPos(coords)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cursor set error: %v\n", err)
			}
		}
	}
	cols, err := termutil.TerminalWidth()
	if err != nil {
		cols = defaultBarWidth
	}
	isFinished := true
	for _, bar := range p.bars {
		if !bar.IsFinished() {
			isFinished = false
		}
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
	p.lastBarsCount = len(p.bars)
	return isFinished
}
