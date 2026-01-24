package cmdutil

import (
    "bytes"
    "fmt"
    "os/exec"
    "strings"
)

// RunGit runs a git command and returns stdout, stderr, exit code, and error.
func RunGit(args ...string) (string, string, int, error) {
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
            code := exitErr.ExitCode()
            return so, se, code, fmt.Errorf("git %v: exit %d: %s", args, code, se)
        }
        return so, se, -1, fmt.Errorf("git %v: %w", args, err)
    }

    return so, se, 0, nil
}
