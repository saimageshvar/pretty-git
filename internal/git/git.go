package git

import (
    "fmt"
    "strings"

    "pretty-git/internal/cmdutil"
)

// encodeKey replaces unsafe git config key characters.
// Git config keys have strict character restrictions. We encode unsafe chars as dots:
// - `/` (slash) → `.` (dot)
// - `_` (underscore) → `.` (dot)  
// - `~` (tilde) → `.` (dot)
// Note: Branch names with different unsafe chars may collide (e.g., "a/b" and "a_b" both encode to "a.b").
// To avoid collisions, branch names should avoid mixing different unsafe characters.
// Branches with `-` (dash) are safe and not encoded.
func encodeKey(s string) string {
    // Replace unsafe characters with dots
    result := strings.ReplaceAll(s, "/", ".")
    result = strings.ReplaceAll(result, "_", ".")
    result = strings.ReplaceAll(result, "~", ".")
    return result
}

// decodeKey cannot fully reverse encodeKey due to collision risk.
// Since we read git config directly by pattern matching on the original branch name,
// we don't need to decode - the branch name is used as the key.
func decodeKey(s string) string {
    // No decoding needed - we use the original branch name as the key when reading.
    return s
}

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
    // if a parent already exists for this child, create a simple backup
    if existing, ok, err := GetParent(child); err != nil {
        return err
    } else if ok {
        // store previous value under pretty-git.parent.backup.<child>
        encodedChild := encodeKey(child)
        _, _, _, err := cmdutil.RunGit("config", "--local", fmt.Sprintf("pretty-git.parent.backup.%s", encodedChild), existing)
        if err != nil {
            return err
        }
    }

    encodedChild := encodeKey(child)
    _, _, _, err := cmdutil.RunGit("config", "--local", fmt.Sprintf("pretty-git.parent.%s", encodedChild), parent)
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
            encodedChild := strings.TrimPrefix(key, prefix)
            child := decodeKey(encodedChild)
            parents[child] = val
        }
    }

    return parents, nil
}

// GetParent returns parent of a child if set
func GetParent(child string) (string, bool, error) {
    encodedChild := encodeKey(child)
    out, _, code, err := cmdutil.RunGit("config", "--local", "--get", fmt.Sprintf("pretty-git.parent.%s", encodedChild))
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
