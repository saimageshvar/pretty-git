package main

import (
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/sai/pretty-git/internal/git"
	ui "github.com/sai/pretty-git/internal/ui"
)

// ansiEscape matches any ANSI SGR escape sequence (e.g. \033[1;35m, \033[0m).
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// wrapBashAnsi wraps every ANSI escape sequence in \x01...\x02 (readline's
// RL_PROMPT_START_IGNORE / RL_PROMPT_END_IGNORE byte markers) so bash correctly
// excludes non-printing sequences from its visible line-length calculation.
//
// NOTE: \[ and \] only work in the static part of PS1. Inside $(...) command
// substitution the output is inserted verbatim, so \[ becomes two literal
// characters. The raw bytes \x01 and \x02 work correctly in both contexts.
func wrapBashAnsi(s string) string {
	return ansiEscape.ReplaceAllStringFunc(s, func(seq string) string {
		return "\x01" + seq + "\x02"
	})
}

// stripAnsi removes all ANSI escape sequences for --no-color output.
func stripAnsi(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// runPrompt outputs a compact git-context segment for embedding in PS1/PROMPT.
//
// Single-line format (default):
//
//	⎇ branchname · short desc…
//
// Two-line format (--newline):
//
//	⎇ branchname · short desc…
//	❯
//
// Nothing is printed when not inside a git repository, so it is safe to add
// unconditionally to any shell prompt — it won't affect non-git directories.
func runPrompt(args []string) {
	fs := flag.NewFlagSet("prompt", flag.ContinueOnError)
	shell   := fs.String("shell",    "",    "shell type for escape wrapping: bash | zsh")
	maxDesc := fs.Int("max-desc",    32,    "max visible chars of description shown (0 = hide description)")
	noColor := fs.Bool("no-color",   false, "strip all color — plain text output")
	newline := fs.Bool("newline",    false, "end with a newline so the shell cursor moves to the next line")
	arrow   := fs.String("arrow",    "❯",   "prompt arrow printed on the second line (only with --newline)")
	fs.Parse(args) //nolint

	// Force true-color output even when stdout is a pipe (as it always is inside
	// a PS1 command substitution). Without this, lipgloss detects no TTY and
	// strips all colour. --no-color overrides this back to plain text.
	if !*noColor {
		lipgloss.SetColorProfile(termenv.TrueColor)
	}

	branch := git.CurrentBranch()
	if branch == "" {
		return // not in a git repo or detached HEAD — output nothing
	}

	// ── Build the git info segment ──────────────────────────────────────────
	branchSt := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent)
	sepSt    := lipgloss.NewStyle().Foreground(ui.ColorDim)
	descSt   := lipgloss.NewStyle().Foreground(ui.ColorDesc).Italic(true)

	seg := branchSt.Render("⎇ " + branch)

	if *maxDesc > 0 {
		desc := strings.TrimSpace(git.GetDescription(branch))
		if desc != "" {
			runes := []rune(desc)
			if len(runes) > *maxDesc {
				desc = string(runes[:*maxDesc-1]) + "…"
			}
			seg += " " + sepSt.Render("·") + " " + descSt.Render(desc)
		}
	}

	// ── Apply color/shell post-processing ──────────────────────────────────
	if *noColor {
		seg = stripAnsi(seg)
	} else if *shell == "bash" {
		seg = wrapBashAnsi(seg)
	}

	// ── Emit ───────────────────────────────────────────────────────────────
	if *newline {
		// Two-line layout: git info on line 1, arrow on line 2.
		// The user's actual typing happens after the arrow on line 2.
		arrowSt := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent)
		arrowSeg := arrowSt.Render(*arrow)
		if *noColor {
			arrowSeg = *arrow
		} else if *shell == "bash" {
			arrowSeg = wrapBashAnsi(arrowSeg)
		}
		fmt.Print(seg + "\n" + arrowSeg + " ")
	} else {
		fmt.Print(seg)
	}
}
