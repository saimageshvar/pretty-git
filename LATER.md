# Later / Nice-to-Have Features

This file captures enhancement ideas and nice-to-have features that can be implemented in future iterations. These are not critical for core functionality but would improve usability and user experience.

---

## Log Command Enhancements

### Pagination/Limits for Large Histories
**Problem:** Currently, `log --all` displays ALL commits in each section without any limits. This can be problematic for repositories with large histories:
- Performance: Fetching and rendering thousands of commits can be slow
- Usability: Overwhelming output that's hard to navigate
- Memory: Large commit lists consume memory

**Potential Solutions:**
- Add `--limit` or `-n` flag to cap commits per section (e.g., `log --all -n 50`)
- Smart defaults: limit to 50-100 commits per section by default
- Show truncation notice: "...and 150 more commits. Use --limit 0 to see all."
- Per-section limits vs. total limit (consider which is more intuitive)

**Example:**
```bash
# Show max 50 commits per section
./pretty-git log --all --limit 50

# Show all commits (no limit)
./pretty-git log --all --limit 0
```

### Merge Source Detection
**Problem:** When another branch gets merged into the current branch, all merged commits appear as "unique to current branch" because they're now in HEAD but not in the parent. Cannot distinguish between:
- Commits authored directly on the current branch
- Commits brought in via merge from other branches

**Potential Solutions:**
- Add `[FROM: branch-name]` tag for commits not originally on this branch
- Add `--show-merge-sources` flag to display which branch commits came from
- Use different color/marker for merged-in commits vs. direct commits
- Parse commit message or git history to detect merge origins

**Example Output:**
```
▲ Current branch only (12)
abc1234  Merge branch 'feature/ui-updates'  [MERGE]
def5678  feat(ui): add new button component  [FROM: feature/ui-updates]
ghi9012  feat(ui): update styling system  [FROM: feature/ui-updates]
jkl3456  feat(auth): add JWT validation
mno7890  feat(auth): add login endpoint
```

---

## Log Command UX Improvements

### Clarify "Parent Branch Only" Section Semantics
**Problem:** The `--all` flag and "Parent branch only" section create cognitive confusion:
- `--all` in git means "all branches", but here means "three-way comparison with parent" (semantic overload)
- Section header "▼ Parent branch only" doesn't explain WHY parent commits are shown or what action is needed
- Users expect `log` to show "what did I do", not "what happened in another branch"
- Mixes "my work" (current branch) with "parent's work" in one view, serving two different questions

**Potential Solutions:**
1. **Rename flag:** Use `--with-parent` or `--compare` instead of `--all` to make comparison explicit
2. **Improve section headers:** Change to more actionable text:
   - Current: `▼ Parent branch only (1)`
   - Better: `▼ New in main since branching (1 behind)`
   - Or: `▼ In parent, not in current (1 commit to sync)`
3. **Separate command:** Create dedicated command for branch comparison:
   - `log` - just your commits (keep current default)
   - `compare-parent` - three-way view with parent
   - `sync-status` - quick ahead/behind summary

**Recommendation:** Quick win is improving section headers to show actionability and branch names explicitly.

---

## Future Enhancements (from initial plan)

### Log Command
- `--graph` flag for ASCII art graph visualization
- `--json` output format for scripting
- Filtering by author, date range, or path
- Integration with `browse` command for interactive log
- Configurable default format (oneline vs multiline)
- GPG signature verification display
- Colorize commit subjects based on conventional commit prefixes (feat:, fix:, etc.)

### General
- Add tests for `internal/git` and `internal/ui` functions
- Branch filtering and grouping by remote/upstream
- Configuration persistence for user preferences
- Support for Git worktrees
- Migration tool for legacy metadata format
- Native packaging with goreleaser and nfpm
