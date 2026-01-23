package cmdutil

import (
    "bytes"
    "fmt"
    "os/exec"
    "strings"
)

// RunGit runs a git command and returns stdout, stderr, and error.
func RunGit(args ...string) (string, string, error) {
    cmd := exec.Command("git", args...)
    var stdout bytes.Buffer
    var stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
    so := strings.TrimSpace(stdout.String())
    se := strings.TrimSpace(stderr.String())
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            return so, se, fmt.Errorf("git %v: exit %d: %s", args, exitErr.ExitCode(), se)
        }
        return so, se, fmt.Errorf("git %v: %w", args, err)
    }

    return so, se, nil
}
