# Code Context — Bug Analysis Verification for `pgit stash`

## Files Retrieved

1. `cmd/pretty-git/stash.go` (lines 1-145) — CLI router; routes `--staged`/`--unstaged`/`--custom` to `StashPush`
2. `internal/git/git.go` (lines 598-700) — `FileStatus`, `StashEntry`, `StashDetail` types; `ListModifiedFiles`, `ListStashes`
3. `internal/git/git.go` (lines 758-880) — `GetStashDetail`, `getStashDetail`, `parseDiff`, skip-set logic, stat computation
4. `internal/git/git.go` (lines 880-960) — `porcelainStatus`, `checkNoCollateral`, `checkStashContent`, `sortedKeys`
5. `internal/git/git.go` (lines 960-1150) — `StashPush` (full implementation with all stashType cases)
6. `internal/ui/stash/browse_model.go` (lines 1-580) — TUI browse model; calls `git.GetStashDetail(ref)` on line ~454 (via `doLoadBrowseDetail`)
7. `memories/pgit-stash.md` — WIP notes confirming `--unstaged` is untested end-to-end

---

## Key Code

### `StashPush` — "staged" case (git.go:1006-1011)
```go
case "staged":
    for path, code := range before {
        if len(code) >= 1 && code[0] != ' ' && code[0] != '?' {
            targetSet[path] = true
        }
    }
    args = []string{"stash", "push", "--staged", "-m", msg}
```

### `getStashDetail` — Skip-set construction (git.go:799-825)
```go
// currentStaged defaults to:  git diff --cached --name-only
// currentUnstaged defaults to: git diff --name-only

// Step 1: HEAD → stash^2, skip = currentStaged
// Step 2: stash^2 → stash,  skip = currentUnstaged
// Step 3: stash^3 (untracked)
```

### `getStashDetail` — Stat line (git.go:858-876)
```go
stat, _ := run("git", "diff", "--stat", "HEAD", ref)
// Parses insertions/deletions from summary line
// This is HEAD vs the FULL WIP tree (stash@{N}), NOT filtered
```

### `browse_model.go` — how browse calls GetStashDetail (line ~454)
```go
func doLoadBrowseDetail(entry git.StashEntry) tea.Cmd {
    return func() tea.Msg {
        d, err := git.GetStashDetail(entry.Ref)  // passes nil, nil for skip sets
        ...
    }
}
```
**Critical:** When called from browse, `GetStashDetail` receives `nil, nil` for skip sets, causing `getStashDetail` to compute them from the *current* working tree via `git diff --cached` and `git diff`.

---

## Bug 1 Verification: `pgit stash --staged` not working

### Proposed Root Cause: "`git stash push --staged` fails on MM files"

**Verdict: PARTIALLY WRONG**

The claim that `git stash push --staged` fails on MM files is **incorrect for modern git** (≥ 2.35). `git stash push --staged` correctly handles MM files: it stashes only the staged portion and leaves unstaged changes intact. It does NOT fail.

However, there **IS** a real bug in `--staged`, and it lives in the **post-stash verification** logic, not in the git command itself.

#### Actual Bug: `checkStashContent` produces false-negative rollbacks

Sequence for `pgit stash --staged` with files `a.go` (staged) and `b.go` (staged):

1. `git stash push --staged -m msg` **succeeds** — stash is created correctly.
2. Post-stash porcelain:
   - `after = {}` (no staged or unstaged changes left, assuming `a.go` and `b.go` were purely staged)
   - `afterStaged = {}`, `afterUnstaged = {}`
3. `checkStashContent(targetSet={"a.go":true, "b.go":true}, afterStaged={}, afterUnstaged={})` is called.
4. Inside, `getStashDetail("stash@{0}", afterStaged={}, afterUnstaged={})` runs:

**Diff 1: `HEAD → stash^2`** — Shows `a.go` (M), `b.go` (M) as staged changes in the stash. Skip set = `currentStaged = {}`. Both files appear. ✓

**Diff 2: `stash^2 → stash`** — For `--staged`, the WIP commit includes the full working tree at stash time. If there were no unstaged changes, diff 2 shows nothing. Skip set = `currentUnstaged = {}`. ✓

Result: detail shows `a.go` and `b.go`. `checkStashContent` sees them in both `targetSet` and `stashedSet`. **Pass.** No rollback.

Now consider with an MM file `a.go` (staged + unstaged changes) and `b.go` (purely staged):

After `git stash push --staged`:
- `a.go` now has only unstaged changes remaining (status " M")
- `b.go` is clean
- `after = {"a.go": " M"}`
- `afterStaged = {}` (code[0]=' ')
- `afterUnstaged = {"a.go": true}` (code[1]='M')

`checkStashContent(targetSet={"a.go":true, "b.go":true}, afterStaged={}, afterUnstaged={"a.go":true})`:

**Diff 1: `HEAD → stash^2`** — Shows `a.go` (M, staged by git) and `b.go` (M). Skip = `{}`. Both appear.

**Diff 2: `stash^2 → stash`** — For `--staged`, the stash WIP tree at `stash@{0}` contains the FULL working directory (including `a.go`'s unstaged changes). The diff `stash^2 → stash` shows `a.go`'s unstaged delta. Skip = `afterUnstaged = {"a.go": true}`. **`a.go` IS skipped.** Good.

Result: detail shows `a.go` (staged) and `b.go` (staged). `checkStashContent` passes. **No rollback. ✓**

#### Where Bug 1 actually manifests

The proposed analysis claims the command fails — but testing shows `git stash push --staged` handles MM files. The **actual** problem with `--staged` is more nuanced:

**Scenario A re-examined**: Two staged files (`a.go`, `b.go`), one unstaged (`c.go`). `pgit stash --staged`:

After stash push --staged:
- `c.go` still has unstaged changes (" M")
- `a.go` and `b.go` are clean (reverted to HEAD)

Post-stash porcelain:
- `after = {"c.go": " M"}`
- `afterStaged = {}`
- `afterUnstaged = {"c.go": true}`

`getStashDetail("stash@{0}", afterStaged={}, afterUnstaged={"c.go":true})`:

- **Diff 1 (`HEAD → stash^2`)**: Shows `a.go`, `b.go` as staged. Skip=`{}`. Both listed. ✓
- **Diff 2 (`stash^2 → stash`)**: The WIP tree contains ALL of the working directory's content. The diff `stash^2 → stash` includes `c.go` (unstaged). Skip=`{"c.go":true}` → `c.go` IS skipped. ✓

Result shows only `a.go` and `b.go`. **This scenario works correctly.**

**But here's the real Bug 1**: The `--staged` stash type fails because `git stash push --staged` can fail in certain git versions or edge cases, and the **error is silently swallowed** — the stash is dropped, but the user sees only a generic error message. Looking at the code:

```go
out, err := exec.Command("git", args...).CombinedOutput()
if err != nil {
    if stashCount() > stashCountBefore {
        exec.Command("git", "stash", "drop").Run()
    }
    return fmt.Errorf("%s", strings.TrimSpace(string(out)))
}
```

This error handling actually looks reasonable — if the stash command fails, it drops any partially-created stash and returns the error.

Actually, after careful re-reading, I believe **Bug 1 as described may not be a real bug**, or if it is, the root cause is different from what was proposed. The `git stash push --staged` command works correctly on MM files in git ≥ 2.35. However, there IS a real issue: **if `--staged` is used on git versions prior to 2.35.0** (which introduced `--staged`), the command will fail entirely with an unrecognized option error. But this is a git version issue, not a pgit code bug.

**Deeper Bug 1 investigation**: The `runStash` routing has a subtle problem. Look at lines 44-56 of `stash.go`:

```go
switch sub {
case "--staged":
    stashType = "staged"
    msgArgs = args[1:]
```

The message is `strings.Join(msgArgs, " ")`. If the user runs `pgit stash --staged`, then `msgArgs = []` and `msg = ""`, which triggers the fallback:

```go
if msg == "" {
    runStashCreate(...); return
}
```

**This means `pgit stash --staged` without a message falls back to the interactive create wizard, NOT a quick stash.** The user may expect it to create a staged stash with a default message, but instead they get the full interactive wizard. This is a UX bug, not a functional bug, but it explains "not working" from the user's perspective if they don't provide a message.

Conversely, `pgit stash` (no args at all) also falls back to the create wizard, which does support staged stash creation. So the "not working" report may simply be that the quick-flag path has no default message behavior.

---

## Bug 2 Verification: Detail pane shows too many files

### Proposed Root Cause Analysis

The analysis identifies two flaws:

1. **Diff 2 skip-set flaw**: A file currently staged (in `currentStaged`) but NOT in `currentUnstaged` won't be skipped by diff 2's skip set, causing false positives.

2. **Stat line flaw**: `git diff --stat HEAD ref` computes stats against the FULL WIP tree, not just the stash's intended content.

**Verdict on flaw 1: CORRECT, but the explanation needs refinement**

The skip-set approach is fundamentally flawed for browsing old stashes. Here's why:

When `GetStashDetail(ref)` is called from browse, `currentStaged` and `currentUnstaged` are computed from the **current working tree** (via `git diff --cached` and `git diff`). These sets represent files that are CURRENTLY modified, not what was modified at stash time.

The comment says:
> "A file still staged/unstaged in the branch was not actually stashed."

This reasoning is ONLY valid when called from `StashPush` (where the current state is right after the stash was created — so anything still modified wasn't stashed). **When called from browse, this reasoning is completely wrong.**

**Specific scenario to validate Bug 2:**

1. User stages `a.go`, runs `pgit stash --staged "msg"`. Stash created successfully.
2. Later, user commits some changes, stages new files. Working tree is now different.
3. User runs `pgit stash list` and browses the old stash.

When viewing the old stash via `GetStashDetail`:
- `currentStaged` = whatever is staged NOW (possibly empty, possibly different files)
- `currentUnstaged` = whatever is unstaged NOW

- **Diff 1 (`HEAD → stash^2`)**: Shows all staged changes AT STASH TIME, skip = currentStaged.
  - If `a.go` is NOT currently staged, it shows up correctly.
  - If `a.go` IS currently staged (e.g., re-staged after the stash), it gets **incorrectly skipped** — the file IS in the stash but is hidden from the detail view.

- **Diff 2 (`stash^2 → stash`)**: Shows WIP-only changes at stash time, skip = currentUnstaged.
  - For `--staged` stashes, the WIP tree contains the FULL working directory. If there were unstaged files at stash time but they aren't currently unstaged, they will NOT be skipped and WILL appear — even though they weren't stashed (they were left in the working tree).

**This confirms Bug 2: files that were never part of the stash can appear in the detail pane.** The skip sets are the wrong filtering mechanism for browsing.

**But also the OPPOSITE problem**: files that ARE in the stash can be HIDDEN if they happen to also currently exist in the same state (staged/unstaged). This is a dual bug — both false positives AND false negatives.

### Verdict on flaw 2: CORRECT

The stat line uses `git diff --stat HEAD ref` which compares HEAD to the COMPLETE WIP tree. For `--staged` and `--custom` stashes, this includes ALL changes (staged + unstaged), not just what was stashed. The insertion/deletion counts will be inflated.

### Additional Root Causes Not Mentioned

**3. `--staged` stash WIP tree contamination**: For `git stash push --staged`, the WIP commit (`stash@{N}`) contains the FULL working tree state at the time, including unstaged changes. This means:
   - `HEAD → stash^2` diff correctly isolates staged changes
   - `stash^2 → stash` diff shows ALL working-tree deviations from the index, which for a `--staged` stash includes unstaged changes that were NEVER stashed
   - The current skip-set logic tries to filter these out, but as analyzed, it's unreliable when browsing old stashes

**4. The `--custom` stash has the same WIP contamination**: `git stash push -- file1 file2` with `--include-untracked` stores the COMPLETE index in `stash^2` and the COMPLETE working tree in `stash@{N}`, even though only the specified files were intended. The pathspec limits what git restores on pop, but the stash commits themselves contain the full snapshot. This causes diff 1 to show ALL staged files and diff 2 to show ALL unstaged files — far more than the custom selection.

**5. `seen` map cross-contamination**: The `seen` map is shared across diff 1, diff 2, and step 3. If a file appears in diff 1 (staged) and also appears in diff 2 (unstaged portion), it will be added to `seen` by diff 1 and then skipped by diff 2. For an MM file that was fully stashed (e.g., `--all` stash), the staged change would appear but the unstaged change would be hidden. This is a minor issue but adds to the confusion.

---

## Scenario Traces

### Scenario A: Two staged (`a.go`, `b.go`), one unstaged (`c.go`), `pgit stash --staged "msg"`

**At stash creation time (StashPush):**
- `targetSet = {"a.go": true, "b.go": true}`
- `git stash push --staged` succeeds
- After stash: `a.go`, `b.go` clean; `c.go` still unstaged (`" M"`)
- `afterStaged = {}`, `afterUnstaged = {"c.go": true}`
- `checkStashContent` calls `getStashDetail("stash@{0}", {}, {"c.go":true})`
  - Diff 1: `HEAD → stash^2` → `a.go`, `b.go`. Skip=`{}`. Both shown. ✓
  - Diff 2: `stash^2 → stash` → `c.go` appears (unstaged change in WIP). Skip=`{"c.go":true}` → skipped. ✓
- Result: `stashedSet = {"a.go", "b.go"}` matches `targetSet`. **Pass.** ✓

**At browse time (later):**
- Suppose `c.go` was committed. Now `currentStaged = {}`, `currentUnstaged = {}`
- `getStashDetail("stash@{0}", nil, nil)` recomputes: `currentStaged = {}`, `currentUnstaged = {}`
  - Diff 1: `HEAD → stash^2` → `a.go`, `b.go`. Skip=`{}`. Both shown.
  - Diff 2: `stash^2 → stash` → `c.go` appears (unstaged at stash time). Skip=`{}` → **NOT skipped! Bug 2 triggers!**
- Result: **3 files shown (`a.go` staged, `b.go` staged, `c.go` unstaged) instead of the 2 that were actually stashed.**

### Scenario B: Three staged (`a.go`, `b.go`, `c.go`), `pgit stash --custom "msg" -- a.go`

**At stash creation time (StashPush):**
- `targetSet = {"a.go": true}`
- `git stash push --include-untracked -m msg -- a.go`
- After stash: `a.go` changes stashed. `b.go` and `c.go` remain staged.
- `before = {"a.go": "M ", "b.go": "M ", "c.go": "M "}`
- `after = {"b.go": "M ", "c.go": "M "}` (a.go removed from staging)
- `afterStaged = {"b.go": true, "c.go": true}`, `afterUnstaged = {}`

**`checkNoCollateral(before, after, {"a.go":true})`:**
- `b.go`: before="M ", after="M " → same. ✓
- `c.go`: before="M ", after="M " → same. ✓
- **Pass.** ✓

**`checkStashContent({"a.go":true}, afterStaged={"b.go":true,"c.go":true}, afterUnstaged={})`:**
- `getStashDetail("stash@{0}", {"b.go":true,"c.go":true}, {})`:
  - Diff 1: `HEAD → stash^2` → shows `a.go`, `b.go`, `c.go` ALL as staged (because `--custom -- a.go` with `--include-untracked` still stores the FULL index as stash^2!)
  - Skip = `currentStaged = {"b.go":true, "c_go":true}` → `b.go` and `c.go` are skipped. Only `a.go` shown. ✓
  - Diff 2: `stash^2 → stash` → differences between index and WIP. For a `--custom` stash of only staged `a.go`, the WIP tree has `a.go` as it was in the working tree. If `a.go` had no unstaged changes, the diff is empty. If it did, it shows those unstaged changes.
  - Skip = `currentUnstaged = {}`. Any file in this diff won't be skipped.
- Result: `stashedSet = {"a.go"}` (assuming no unstaged delta). Matches `targetSet`. **Pass.** ✓

**At browse time (later):**
- Suppose `b.go` and `c.go` were committed. `currentStaged = {}`, `currentUnstaged = {}`
- `getStashDetail` recomputes:
  - Diff 1: `HEAD → stash^2` → `a.go`, `b.go`, `c.go` ALL shown (nothing in skip set). **Bug 2 triggers! 3 files shown instead of 1.**
  - Diff 2: `stash^2 → stash` → if any unstaged changes existed at stash time, they'd appear too (unfiltered).
- Result: **Far more files than just `a.go`.**

### Scenario C: MM file (`a.go` both staged and unstaged), `pgit stash --staged "msg"`

**At stash creation time:**
- `targetSet = {"a.go": true}` (code[0] = 'M')
- `git stash push --staged -m msg` succeeds
- After stash: `a.go` has only unstaged changes (" M")
- `afterUnstaged = {"a.go": true}`, `afterStaged = {}`

**`checkStashContent({"a.go":true}, {}, {"a.go":true})`:**
- `getStashDetail("stash@{0}", {}, {"a.go":true})`:
  - Diff 1: `HEAD → stash^2` → `a.go` (M). Skip=`{}`. Shown as staged. ✓
  - Diff 2: `stash^2 → stash` → `a.go` (unstaged delta). Skip=`{"a.go":true}`. **Skipped!** ✓
- Result: `stashedSet = {"a.go"}`. Matches `targetSet`. **Pass.** ✓

**At browse time (later when `a.go` is clean):**
- `currentStaged = {}`, `currentUnstaged = {}`
- Diff 2: `stash^2 → stash` → `a.go` (unstaged delta). Skip=`{}`. **NOT skipped!** Bug 2 triggers: `a.go` appears TWICE (once as staged "M " from diff 1, once as unstaged " M" from diff 2).
  - BUT: `seen` map prevents true duplicates. `a.go` was already added by diff 1, so diff 2's appearance is skipped by `seen`. Result: `a.go` shown once as staged. ✓
  
  Wait — but this means `a.go` is shown as **staged only**, hiding the unstaged portion. For a `--staged` stash, this is actually **correct behavior** — only the staged portion was stashed. So the `seen` map accidentally helps here, but it's masking the underlying skip-set bug.

---

## Assessment: What Needs to Change

### Bug 1: `pgit stash --staged` not working

**Root cause (revised):** The proposed analysis is *incorrect* — `git stash push --staged` does NOT fail on MM files. The actual issues are:

1. **UX/Missing feature**: `pgit stash --staged` (no message) falls back to the interactive wizard rather than performing a quick stash with a default message. This is likely what users mean by "not working."

2. **Potential post-verification false rollback**: While the `--staged` case passes `checkStashContent` in most common scenarios (as traced above), there's a fragile dependency on `getStashDetail` correctly identifying which files are in the stash. If the skip sets are wrong (which they can be in edge cases), `checkStashContent` will either:
   - See unexpected files → trigger false rollback
   - Not see expected files → trigger false rollback
   
   This could cause the stash to be popped back (rolled back) even though `git stash push --staged` succeeded correctly. **This is a real risk.**

**Fix needed:**
- Add a default message for `pgit stash --staged` (and `--unstaged`) when no message is provided, OR make the interactive wizard pre-select the `--staged` type.
- Consider whether `checkStashContent` is actually providing value for `--staged` stashes, or whether it's creating false rollbacks. The reactive `stash drop + error` for failed commands is fine, but the proactive content verification is fragile.

### Bug 2: Detail pane shows too many files

**Root cause (confirmed + expanded):**

The proposed analysis is **correct** that the skip sets cause false positives. But the analysis is **incomplete**:

1. **Skip sets are based on current working-tree state, which diverges from stash-time state.** When browsing old stashes, `currentStaged` and `currentUnstaged` represent what's modified NOW, not what was modified at stash time. This causes:
   - **False positives**: Files that appear in the stash diffs but weren't actually stashed get shown (because they're no longer modified and thus not in the skip sets).
   - **False negatives**: Files that ARE in the stash get hidden (because they happen to be in the same modified state NOW as they were at stash time).

2. **`--custom` stash WIP contamination**: For `git stash push -- file1`, the stash commits still contain the FULL index and FULL working tree. The skip-set approach is the only thing preventing all staged files from appearing, but it's unreliable.

3. **Stat inflation**: `git diff --stat HEAD stash@{N}` includes ALL changes (staged + unstaged + untracked), not just what was stashed.

**Fix needed:**
- **Option A (Best)**: Store the stash type (`staged`/`unstaged`/`all`/`custom`) and the list of target files in the stash message or a metadata sidecar. When browsing, use this metadata to filter the display rather than relying on skip sets.
- **Option B (Pragmatic)**: For browse mode (`GetStashDetail` called without skip sets), use a different strategy:
  - For `--staged` type stashes: only show diff 1 files (HEAD → stash^2), skip diff 2 entirely.
  - For `--unstaged` type stashes: only show diff 2 files (stash^2 → stash), skip diff 1.
  - For `--all` type: show both.
  - For `--custom` type: intersect both diffs with the pathspec.
  - This requires knowing the stash type, which isn't currently encoded.
- **Option C (Minimal)**: Remove the skip-set logic entirely from the browse path (pass empty skip sets). This shows ALL files in the stash commits, which is more than desirable but at least is consistent and doesn't hide legitimate stashed files. Then fix the stat line to only count the files shown in the detail.
- **Stat line fix (required regardless)**: Replace `git diff --stat HEAD ref` with a targeted stat that only accounts for the files shown in the detail, not the full WIP diff.

### Summary Table

| Bug | Proposed Analysis | Verdict | Gap |
|-----|-------------------|---------|-----|
| Bug 1 | `git stash push --staged` fails on MM files | **Incorrect** — `--staged` handles MM files fine | Real issue: missing default message → falls back to wizard; fragile `checkStashContent` can cause false rollbacks |
| Bug 2: skip sets | Files in `currentStaged` but not `currentUnstaged` leak into diff 2 | **Correct** but incomplete | Also causes false negatives (files hidden); stat line inflation; `--custom` stash full-index contamination; fundamental problem is using current working-tree state to filter historical stash data |
| Bug 2: stat line | `HEAD vs ref` includes all changes | **Correct** | Should be derived from filtered file list |