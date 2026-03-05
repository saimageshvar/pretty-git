package update

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	checkTTL = 24 * time.Hour

	latestReleaseURL = "https://api.github.com/repos/saimageshvar/pretty-git/releases/latest"
	installerCommand = "curl -sSL https://raw.githubusercontent.com/saimageshvar/pretty-git/main/install.sh | sudo bash"
)

var httpClient = &http.Client{Timeout: 3 * time.Second}

type semver struct {
	major int
	minor int
	patch int
}

type cacheData struct {
	LastCheckedAt string `json:"last_checked_at"`
	LatestVersion string `json:"latest_version"`
	Source        string `json:"source"`
	LastError     string `json:"last_error,omitempty"`
}

// MaybeNotifyAndUpdate checks GitHub releases and prompts users to update.
// It is non-blocking for command execution on network/cache errors.
func MaybeNotifyAndUpdate(ctx context.Context, command, currentVersion string) {
	if os.Getenv("PGIT_NO_UPDATE_CHECK") == "1" {
		return
	}
	if !shouldCheckCommand(command) || !isInteractive() {
		return
	}

	current, ok := parseSemver(currentVersion)
	if !ok {
		return
	}

	latestTag, err := getLatestVersion(ctx)
	if err != nil || latestTag == "" {
		return
	}

	latest, ok := parseSemver(latestTag)
	if !ok || compareSemver(latest, current) <= 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "pgit: update available %s (current %s)\n", latestTag, currentVersion)
	if !confirmUpdate(os.Stdin, os.Stderr) {
		return
	}

	if err := runInstaller(); err != nil {
		fmt.Fprintf(os.Stderr, "pgit: update failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "pgit: run manually: %s\n", installerCommand)
		return
	}

	fmt.Fprintln(os.Stderr, "pgit: update complete.")
}

func shouldCheckCommand(command string) bool {
	switch command {
	case "", "prompt":
		return false
	default:
		return true
	}
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))
}

func getLatestVersion(ctx context.Context) (string, error) {
	path, err := cachePath()
	if err != nil {
		return fetchLatestVersion(ctx)
	}

	cache, ok := readCache(path)
	if ok && cache.LastCheckedAt != "" {
		lastCheckedAt, err := time.Parse(time.RFC3339, cache.LastCheckedAt)
		if err == nil && time.Since(lastCheckedAt) < checkTTL && cache.LatestVersion != "" {
			return cache.LatestVersion, nil
		}
	}

	latest, err := fetchLatestVersion(ctx)
	if err != nil {
		if ok && cache.LatestVersion != "" {
			return cache.LatestVersion, nil
		}
		return "", err
	}

	_ = writeCache(path, cacheData{
		LastCheckedAt: time.Now().UTC().Format(time.RFC3339),
		LatestVersion: latest,
		Source:        "github_release_latest",
	})

	return latest, nil
}

func fetchLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pgit-update-check")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github returned %s", resp.Status)
	}

	var payload struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	if payload.Draft || payload.Prerelease {
		return "", errors.New("latest release is not stable")
	}
	if payload.TagName == "" {
		return "", errors.New("missing tag_name in release response")
	}
	return payload.TagName, nil
}

func cachePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "pgit", "update.json"), nil
}

func readCache(path string) (cacheData, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return cacheData{}, false
	}
	var c cacheData
	if err := json.Unmarshal(b, &c); err != nil {
		return cacheData{}, false
	}
	return c, true
}

func writeCache(path string, c cacheData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func confirmUpdate(in io.Reader, out io.Writer) bool {
	fmt.Fprint(out, "Update now? [Y/n]: ")
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	switch answer {
	case "", "y", "yes":
		return true
	default:
		return false
	}
}

func runInstaller() error {
	cmd := exec.Command("bash", "-lc", installerCommand)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func parseSemver(v string) (semver, bool) {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return semver{}, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return semver{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return semver{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return semver{}, false
	}
	return semver{major: major, minor: minor, patch: patch}, true
}

func compareSemver(a, b semver) int {
	if a.major != b.major {
		if a.major < b.major {
			return -1
		}
		return 1
	}
	if a.minor != b.minor {
		if a.minor < b.minor {
			return -1
		}
		return 1
	}
	if a.patch != b.patch {
		if a.patch < b.patch {
			return -1
		}
		return 1
	}
	return 0
}
