package git

import (
    "fmt"
    "strings"

    "pretty-git/internal/cmdutil"
)

// ListBranches returns local branch names
func ListBranches() ([]string, error) {
    out, _, _, err := cmdutil.RunGit("for-each-ref", "refs/heads", "--format=%(refname:short)")
    if err != nil {
        return nil, err
    }

    if out == "" {
        return []string{}, nil
    }

    lines := strings.Split(out, "\n")
    for i := range lines {
        lines[i] = strings.TrimSpace(lines[i])
    }
    return lines, nil
}

// GetCurrentBranch returns the current branch name or error if detached
func GetCurrentBranch() (string, error) {
    out, _, _, err := cmdutil.RunGit("rev-parse", "--abbrev-ref", "HEAD")
    if err != nil {
        return "", err
    }
    out = strings.TrimSpace(out)
    if out == "HEAD" {
        return "", fmt.Errorf("detached HEAD")
    }
    return out, nil
}

// SetParent writes pretty-git.parent.<child> in local git config
func SetParent(child, parent string) error {
    _, _, _, err := cmdutil.RunGit("config", "--local", fmt.Sprintf("pretty-git.parent.%s", child), parent)
    return err
}

// AllParents reads all pretty-git.parent.* entries and returns map[child]=parent
func AllParents() (map[string]string, error) {
    out, stderr, code, err := cmdutil.RunGit("config", "--local", "--get-regexp", "^pretty-git\\.parent\\.")
    if err != nil {
        // git returns exit code 1 when no matches are found; treat as empty
        if code == 1 {
            return map[string]string{}, nil
        }
        // if stderr contains known messages also tolerate
        if strings.Contains(stderr, "no matching") || strings.Contains(stderr, "No such file or directory") {
            return map[string]string{}, nil
        }
        return nil, err
    }

    parents := map[string]string{}
    if out == "" {
        return parents, nil
    }

    lines := strings.Split(strings.TrimSpace(out), "\n")
    for _, l := range lines {
        parts := strings.Fields(l)
        if len(parts) < 2 {
            continue
        }
        key := parts[0]
        val := parts[1]
        // key is pretty-git.parent.<child>
        const prefix = "pretty-git.parent."
        if strings.HasPrefix(key, prefix) {
            child := strings.TrimPrefix(key, prefix)
            parents[child] = val
        }
    }

    return parents, nil
}

// GetParent returns parent of a child if set
func GetParent(child string) (string, bool, error) {
    out, _, code, err := cmdutil.RunGit("config", "--local", "--get", fmt.Sprintf("pretty-git.parent.%s", child))
    if err != nil {
        if code == 1 {
            return "", false, nil
        }
        return "", false, err
    }
    return strings.TrimSpace(out), true, nil
}

// CheckoutBranch performs git checkout. If create is true, behaves like 'git checkout -b <branch> [parent]'
func CheckoutBranch(branch string, create bool, parent string) error {
    if create {
        if parent != "" {
            _, _, _, err := cmdutil.RunGit("checkout", "-b", branch, parent)
            return err
        }
        _, _, _, err := cmdutil.RunGit("checkout", "-b", branch)
        return err
    }
    _, _, _, err := cmdutil.RunGit("checkout", branch)
    return err
}
