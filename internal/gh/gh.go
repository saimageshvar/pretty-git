package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PR represents a pull request from GitHub.
type PR struct {
	Number         int       `json:"number"`
	Title          string    `json:"title"`
	State          string    `json:"state"`
	Author         string    `json:"author"`
	BaseRef        string    `json:"baseRefName"`
	HeadRef        string    `json:"headRefName"`
	CreatedAt      time.Time `json:"createdAt"`
	ReviewDecision string    `json:"reviewDecision"`
	Comments       int       `json:"comments"`
	Additions      int       `json:"additions"`
	Deletions      int       `json:"deletions"`
	ChangedFiles   int       `json:"changedFiles"`
	URL            string    `json:"url"`
	Approvals      int       `json:"approvals"` // count of APPROVED reviews
}

// prJSON is the raw JSON structure from gh pr list.
type prJSON struct {
	Number         int            `json:"number"`
	Title          string         `json:"title"`
	State          string         `json:"state"`
	Author         prAuthor       `json:"author"`
	BaseRefName    string         `json:"baseRefName"`
	HeadRefName    string         `json:"headRefName"`
	CreatedAt      string         `json:"createdAt"`
	ReviewDecision string         `json:"reviewDecision"`
	Comments       []interface{}  `json:"comments"`
	Additions      int            `json:"additions"`
	Deletions      int            `json:"deletions"`
	ChangedFiles   int            `json:"changedFiles"`
	URL            string         `json:"url"`
	LatestReviews  []prReview     `json:"latestReviews"`
}

type prReview struct {
	State  string `json:"state"`
	Author string `json:"author"`
}

type prAuthor struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

// WorkflowRun represents a GitHub Actions workflow run.
type WorkflowRun struct {
	ID           int64     `json:"databaseId"`
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion"`
	DisplayTitle string    `json:"displayTitle"`
	HeadBranch   string    `json:"headBranch"`
	WorkflowName string    `json:"workflowName"`
	CreatedAt    time.Time `json:"createdAt"`
}

// IsAvailable returns true if gh CLI is installed and authenticated.
func IsAvailable() bool {
	cmd := exec.Command("gh", "auth", "status")
	err := cmd.Run()
	return err == nil
}

// ListMyPRs fetches PRs authored by the current user.
func ListMyPRs(limit int) ([]PR, error) {
	if limit <= 0 {
		limit = 30
	}
	fields := "number,title,state,author,baseRefName,headRefName,createdAt,reviewDecision,comments,additions,deletions,changedFiles,url,latestReviews"
	cmd := exec.Command("gh", "pr", "list",
		"--author", "@me",
		"--state", "all",
		"--json", fields,
		"--limit", fmt.Sprintf("%d", limit),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh pr list: %w", err)
	}
	return parsePRs(out)
}

// ListReviewRequested fetches PRs awaiting the current user's review.
func ListReviewRequested(limit int) ([]PR, error) {
	if limit <= 0 {
		limit = 30
	}
	fields := "number,title,state,author,baseRefName,headRefName,createdAt,reviewDecision,comments,additions,deletions,changedFiles,url,latestReviews"
	cmd := exec.Command("gh", "pr", "list",
		"--search", "review-requested:@me",
		"--state", "open",
		"--json", fields,
		"--limit", fmt.Sprintf("%d", limit),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh pr list: %w", err)
	}
	return parsePRs(out)
}

// parsePRs converts JSON output from gh pr list into []PR.
func parsePRs(data []byte) ([]PR, error) {
	var raw []prJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse PR JSON: %w", err)
	}

	var prs []PR
	for _, r := range raw {
		createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
		author := r.Author.Login
		if r.Author.Name != "" {
			author = r.Author.Name
		}
		// Count approvals from latestReviews
		approvals := 0
		for _, review := range r.LatestReviews {
			if review.State == "APPROVED" {
				approvals++
			}
		}
		prs = append(prs, PR{
			Number:         r.Number,
			Title:          r.Title,
			State:          r.State,
			Author:         author,
			BaseRef:        r.BaseRefName,
			HeadRef:        r.HeadRefName,
			CreatedAt:      createdAt,
			ReviewDecision: r.ReviewDecision,
			Comments:       len(r.Comments),
			Additions:      r.Additions,
			Deletions:      r.Deletions,
			ChangedFiles:   r.ChangedFiles,
			URL:            r.URL,
			Approvals:      approvals,
		})
	}
	return prs, nil
}

// OpenInBrowser opens the PR URL in the default browser.
func OpenInBrowser(url string) error {
	cmd := exec.Command("gh", "browse", url)
	return cmd.Run()
}

// DaysOpen returns the number of days since the PR was created.
func DaysOpen(createdAt time.Time) int {
	return int(time.Since(createdAt).Hours() / 24)
}

// StateIcon returns an icon for the PR state.
func StateIcon(state string) string {
	switch strings.ToUpper(state) {
	case "OPEN":
		return "●"
	case "MERGED":
		return "✓"
	case "CLOSED":
		return "✗"
	default:
		return "?"
	}
}

// ReviewStatus returns a human-readable review status.
func ReviewStatus(decision string) string {
	switch strings.ToUpper(decision) {
	case "APPROVED":
		return "✓ approved"
	case "CHANGES_REQUESTED":
		return "✗ changes requested"
	case "REVIEW_REQUIRED":
		return "⏳ review required"
	default:
		return ""
	}
}