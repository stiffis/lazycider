package kitty

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type DrawOptions struct {
	ImageWidth  int
	ImageHeight int
	TermWidth   int
	TermHeight  int
	PanelX      int
	PanelY      int
	PanelW      int
	PanelH      int
	Clear       bool
}

func Clear() error {
	if !IsKitty() {
		return nil
	}
	if _, err := exec.LookPath("kitty"); err != nil {
		return nil
	}
	cmd := exec.Command("kitty", "+kitten", "icat", "--clear")
	return cmd.Run()
}

func Draw(path string, opts DrawOptions) error {
	if !IsKitty() {
		return fmt.Errorf("kitty terminal not detected")
	}
	if _, err := exec.LookPath("kitty"); err != nil {
		return fmt.Errorf("kitty binary not found")
	}

	placeW, placeH := ComputeCoverPlacementSize(opts.TermWidth, opts.PanelW, opts.PanelH, opts.ImageWidth, opts.ImageHeight)
	if placeW <= 0 || placeH <= 0 {
		return fmt.Errorf("invalid placement size")
	}

	offsetX := opts.PanelX + (opts.PanelW-placeW)/2
	offsetY := opts.PanelY

	if opts.Clear {
		_ = Clear()
	}

	place := fmt.Sprintf("%dx%d@%dx%d", placeW, placeH, offsetX, offsetY)
	args := []string{
		"+kitten", "icat",
		"--stdin=no",
		"--transfer-mode=file",
		"--place", place,
	}
	if UseUnicodePlaceholders() {
		args = append(args, "--unicode-placeholder")
		windowSize := fmt.Sprintf("%d,%d,%d,%d", opts.TermWidth, opts.TermHeight, opts.TermWidth*10, opts.TermHeight*20)
		args = append(args, "--use-window-size", windowSize)
	}
	args = append(args, path)

	cmd := exec.Command("kitty", args...)
	cmd.Stdout = os.Stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("icat draw failed: %s", msg)
	}

	return nil
}
