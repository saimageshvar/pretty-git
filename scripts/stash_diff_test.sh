#!/usr/bin/env bash
# Stash-internal diff verification — 10 test cases
# Tests that stash^1, stash^2, stash references produce correct diffs
# regardless of current branch, commits, or file state changes.

REPO=~/projects/pg-test
PASS=0
FAIL=0

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
RESET='\033[0m'

pass() { echo -e "  ${GREEN}✓ $1${RESET}"; ((PASS++)); }
fail() { echo -e "  ${RED}✗ $1${RESET}"; ((FAIL++)); }

reset_repo() {
    cd "$REPO"
    git stash list 2>/dev/null | while read -r _; do git stash drop 2>/dev/null; done
    git checkout -- . 2>/dev/null
    git clean -fd 2>/dev/null | grep -v "^Removing \.serena" > /dev/null 2>&1
    git reset HEAD -- . 2>/dev/null
    git checkout -- . 2>/dev/null
    # Reset all files to initial content
    echo "hello" > a.go
    echo "world" > b.go
    echo "foo" > c.go
    git add -A
    git commit -m "clean state" --allow-empty 2>/dev/null
}

# ── Test 1: Staged-only stash — stash^1 vs stash^2 shows only staged files ──
run_test "1: Staged stash — diff stash^1 vs stash^2 shows only staged files"
reset_repo
echo "// staged change a" >> a.go && git add a.go
echo "// staged change b" >> b.go && git add b.go
echo "// unstaged change c" >> c.go
git stash push --staged -m "test1-staged"

STASH_REF=$(git stash list --format="%gd" | head -1)
DIFF_12=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)

if echo "$DIFF_12" | grep -q "^a.go$" && echo "$DIFF_12" | grep -q "^b.go$"; then
    pass "stash^1 vs stash^2 contains a.go and b.go (staged files)"
else
    fail "stash^1 vs stash^2 should contain a.go and b.go, got: $DIFF_12"
fi

if echo "$DIFF_12" | grep -q "c.go"; then
    fail "stash^1 vs stash^2 should NOT contain c.go (unstaged)"
else
    pass "stash^1 vs stash^2 does NOT contain c.go (unstaged)"
fi

echo "// unstaged c" >> c.go  # make more changes
DIFF_12_AFTER=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)
if [ "$DIFF_12" = "$DIFF_12_AFTER" ]; then
    pass "diff is unchanged after modifying working tree"
else
    fail "diff CHANGED after modifying working tree — should be immutable!"
fi

# ── Test 2: Unstaged stash — stash^2 vs stash shows only unstaged files ──
run_test "2: Unstaged stash — diff stash^2 vs stash shows only unstaged files"
reset_repo
echo "// staged a" >> a.go && git add a.go
echo "// unstaged b" >> b.go
git stash push --keep-index -m "test2-unstaged"

STASH_REF=$(git stash list --format="%gd" | head -1)
DIFF_23=$(git diff --name-only "$STASH_REF^2" "$STASH_REF" 2>/dev/null)

if echo "$DIFF_23" | grep -q "^b.go$"; then
    pass "stash^2 vs stash contains b.go (unstaged)"
else
    fail "stash^2 vs stash should contain b.go, got: $DIFF_23"
fi

if echo "$DIFF_23" | grep -q "^a.go$"; then
    fail "stash^2 vs stash should NOT contain a.go (staged, kept by --keep-index)"
else
    pass "stash^2 vs stash does NOT contain a.go (only staged changes, no unstaged delta)"
fi

# Verify a.go is still staged (kept by --keep-index)
STATUS=$(git status --porcelain a.go)
if echo "$STATUS" | grep -q "^M"; then
    pass "a.go is still staged after --keep-index stash"
else
    fail "a.go should still be staged after --keep-index, got: $STATUS"
fi

# ── Test 3: All stash — both diffs together show everything ──
run_test "3: All stash (include-untracked) — full diff shows all files"
reset_repo
echo "// staged a" >> a.go && git add a.go
echo "// unstaged b" >> b.go
echo "// untracked new" > new.txt
git stash push --include-untracked -m "test3-all"

STASH_REF=$(git stash list --format="%gd" | head -1)
DIFF_12=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)
DIFF_23=$(git diff --name-only "$STASH_REF^2" "$STASH_REF" 2>/dev/null)
UNTRACKED=$(git ls-tree --name-only "$STASH_REF^3" 2>/dev/null)

if echo "$DIFF_12" | grep -q "^a.go$"; then
    pass "stash^1 vs stash^2 contains a.go (staged)"
else
    fail "stash^1 vs stash^2 should contain a.go"
fi

if echo "$DIFF_23" | grep -q "^b.go$"; then
    pass "stash^2 vs stash contains b.go (unstaged)"
else
    fail "stash^2 vs stash should contain b.go"
fi

if echo "$UNTRACKED" | grep -q "^new.txt$"; then
    pass "stash^3 contains new.txt (untracked)"
else
    fail "stash^3 should contain new.txt, got: $UNTRACKED"
fi

# ── Test 4: Staged stash after switching branches ──
run_test "4: Staged stash — diff stable after switching branches"
reset_repo
echo "// staged a" >> a.go && git add a.go
echo "// staged b" >> b.go && git add b.go
git stash push --staged -m "test4-staged"

STASH_REF=$(git stash list --format="%gd" | head -1)
DIFF_BEFORE=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)

# Create a new branch, modify files, commit
git checkout -b test-branch 2>/dev/null
echo "// branch change" >> a.go && git add a.go && git commit -m "branch commit" --allow-empty 2>/dev/null
echo "// more changes" >> b.go && git add b.go && git commit -m "another commit" --allow-empty 2>/dev/null

DIFF_AFTER=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)

if [ "$DIFF_BEFORE" = "$DIFF_AFTER" ]; then
    pass "diff is identical after branch switch and commits"
else
    fail "diff CHANGED after branch switch — should be immutable!"
    echo "  Before: $DIFF_BEFORE"
    echo "  After:  $DIFF_AFTER"
fi

git checkout master 2>/dev/null

# ── Test 5: Staged stash after deleting files ──
run_test "5: Staged stash — diff stable after deleting files"
reset_repo
echo "// staged a" >> a.go && git add a.go
echo "// staged b" >> b.go && git add b.go
git stash push --staged -m "test5-staged"

STASH_REF=$(git stash list --format="%gd" | head -1)
DIFF_BEFORE=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)

# Delete files that were in the stash
rm -f a.go b.go
git add -A && git commit -m "deleted files" --allow-empty 2>/dev/null

DIFF_AFTER=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)

if [ "$DIFF_BEFORE" = "$DIFF_AFTER" ]; then
    pass "diff is identical after deleting files"
else
    fail "diff CHANGED after deleting files — should be immutable!"
    echo "  Before: $DIFF_BEFORE"
    echo "  After:  $DIFF_AFTER"
fi

# Restore files for next test
git checkout HEAD~1 -- a.go b.go 2>/dev/null || { echo "hello" > a.go; echo "world" > b.go; }
git add -A && git commit -m "restored" --allow-empty 2>/dev/null

# ── Test 6: Custom (pathspec) stash — stash^2 contains full index ──
run_test "6: Custom stash — stash^2 contains ALL staged files, not just selected"
reset_repo
echo "// staged a" >> a.go && git add a.go
echo "// staged b" >> b.go && git add b.go
echo "// staged c" >> c.go && git add c.go
git stash push --include-untracked -m "test6-custom" -- a.go

STASH_REF=$(git stash list --format="%gd" | head -1)

# stash^2 should contain ALL staged files (not just a.go)
DIFF_12=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)

echo "  stash^1 vs stash^2: $DIFF_12"

if echo "$DIFF_12" | grep -q "^a.go$"; then
    pass "stash^2 contains a.go (selected file)"
else
    fail "stash^2 should contain a.go"
fi

if echo "$DIFF_12" | grep -q "^b.go$"; then
    pass "stash^2 ALSO contains b.go (non-selected staged file — this is why we need the prefix!)"
else
    fail "stash^2 should contain b.go (full index snapshot)"
fi

if echo "$DIFF_12" | grep -q "^c.go$"; then
    pass "stash^2 ALSO contains c.go (non-selected staged file)"
else
    fail "stash^2 should contain c.go (full index snapshot)"
fi

# ── Test 7: Custom stash — filtering by target files gives correct result ──
run_test "7: Custom stash — filtering diff to target files shows only a.go"
# Continuing from test 6, we know stash^2 has a.go, b.go, c.go
# But the [pgit:custom:a.go] prefix tells us only a.go was selected

# Simulate what GetStashDetail would do for custom type:
# Use both diffs but filter to target files
DIFF_23=$(git diff --name-only "$STASH_REF^2" "$STASH_REF" 2>/dev/null)
ALL_FILES=$( (echo "$DIFF_12"; echo "$DIFF_23") | sort -u )

# Filter to target files (a.go only)
FILTERED=$(echo "$ALL_FILES" | grep "^a.go$" || true)

if [ "$FILTERED" = "a.go" ]; then
    pass "Filtering all diffs to target {a.go} gives correct result"
else
    fail "Filtering should give only a.go, got: $FILTERED"
fi

NOT_FILTERED=$(echo "$ALL_FILES" | grep "^b.go$" || true)
if [ -z "$NOT_FILTERED" ]; then
    pass "b.go correctly excluded when filtering to a.go"
else
    fail "b.go should NOT appear when filtering to a.go only"
fi

# ── Test 8: Unstaged stash — diff stable after HEAD changes ──
run_test "8: Unstaged stash — diff stable after new commits"
reset_repo
echo "// staged a" >> a.go && git add a.go
echo "// unstaged b" >> b.go
git stash push --keep-index -m "test8-unstaged"

STASH_REF=$(git stash list --format="%gd" | head -1)
DIFF_BEFORE=$(git diff --name-only "$STASH_REF^2" "$STASH_REF" 2>/dev/null)

# Make several commits that move HEAD
echo "// commit1" >> a.go && git add a.go && git commit -m "commit1" --allow-empty 2>/dev/null
echo "// commit2" >> a.go && git add a.go && git commit -m "commit2" --allow-empty 2>/dev/null
echo "// commit3" >> a.go && git add a.go && git commit -m "commit3" --allow-empty 2>/dev/null

DIFF_AFTER=$(git diff --name-only "$STASH_REF^2" "$STASH_REF" 2>/dev/null)

if [ "$DIFF_BEFORE" = "$DIFF_AFTER" ]; then
    pass "stash^2 vs stash diff is identical after 3 new commits"
else
    fail "stash^2 vs stash diff CHANGED after commits — should be immutable!"
    echo "  Before: $DIFF_BEFORE"
    echo "  After:  $DIFF_AFTER"
fi

# ── Test 9: All stash — full diff (stash^1 vs stash) shows everything ──
run_test "9: All stash — diff stash^1 vs stash shows all tracked changes"
reset_repo
echo "// staged a" >> a.go && git add a.go
echo "// unstaged b" >> b.go
echo "// staged c" >> c.go && git add c.go
echo "// untracked" > new.txt
git stash push --include-untracked -m "test9-all"

STASH_REF=$(git stash list --format="%gd" | head -1)
DIFF_FULL=$(git diff --name-only "$STASH_REF^1" "$STASH_REF" 2>/dev/null)
DIFF_12=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)
DIFF_23=$(git diff --name-only "$STASH_REF^2" "$STASH_REF" 2>/dev/null)
UNTRACKED=$(git ls-tree --name-only "$STASH_REF^3" 2>/dev/null)

echo "  Full diff (stash^1 vs stash): $DIFF_FULL"
echo "  Staged diff (stash^1 vs stash^2): $DIFF_12"
echo "  Unstaged diff (stash^2 vs stash): $DIFF_23"
echo "  Untracked (stash^3): $UNTRACKED"

# Full diff should show a.go, b.go, c.go
FULL_OK=true
for f in a.go b.go c.go; do
    if ! echo "$DIFF_FULL" | grep -q "^$f$"; then
        fail "Full diff should contain $f"
        FULL_OK=false
    fi
done
if $FULL_OK; then
    pass "Full diff contains a.go, b.go, c.go"
fi

if echo "$UNTRACKED" | grep -q "^new.txt$"; then
    pass "stash^3 contains new.txt (untracked)"
else
    fail "stash^3 should contain new.txt"
fi

# ── Test 10: Old HEAD reference would give wrong results ──
run_test "10: OLD approach (diff HEAD stash^2) gives WRONG results after commits"
reset_repo
echo "// staged a" >> a.go && git add a.go
echo "// staged b" >> b.go && git add b.go
git stash push --staged -m "test10-staged"

STASH_REF=$(git stash list --format="%gd" | head -1)

# Correct approach: stash^1 vs stash^2
CORRECT_DIFF=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)

# Wrong approach (old code): HEAD vs stash^2
WRONG_DIFF_BEFORE=$(git diff --name-only HEAD "$STASH_REF^2" 2>/dev/null)

# These should be the same right after stash creation
if [ "$CORRECT_DIFF" = "$WRONG_DIFF_BEFORE" ]; then
    pass "Right after stash: HEAD vs stash^2 matches stash^1 vs stash^2 (expected)"
else
    fail "Right after stash: diffs differ (unexpected)"
    echo "  Correct: $CORRECT_DIFF"
    echo "  Wrong:   $WRONG_DIFF_BEFORE"
fi

# Now make commits that move HEAD
echo "// more commits" >> a.go && git add a.go && git commit -m "move head" --allow-empty 2>/dev/null
echo "// more commits" >> b.go && git add b.go && git commit -m "move head again" --allow-empty 2>/dev/null

# Correct approach is unchanged
CORRECT_AFTER=$(git diff --name-only "$STASH_REF^1" "$STASH_REF^2" 2>/dev/null)

# Wrong approach (old code) now includes DIFFERENT results
WRONG_DIFF_AFTER=$(git diff --name-only HEAD "$STASH_REF^2" 2>/dev/null)

if [ "$CORRECT_DIFF" = "$CORRECT_AFTER" ]; then
    pass "stash^1 vs stash^2 is STILL correct after commits (immutable)"
else
    fail "stash^1 vs stash^2 changed (shouldn't happen!)"
fi

if [ "$WRONG_DIFF_BEFORE" != "$WRONG_DIFF_AFTER" ]; then
    pass "OLD approach (HEAD vs stash^2) CHANGED after commits — proving it was WRONG"
    echo "  Before commits: $WRONG_DIFF_BEFORE"
    echo "  After commits:   $WRONG_DIFF_AFTER"
else
    fail "OLD approach (HEAD vs stash^2) did NOT change — coincidental, still unreliable"
fi

# ── Summary ──
echo ""
echo -e "${CYAN}════════════════════════════════════════════════════════${RESET}"
total=$((PASS + FAIL))
echo -e "  Results: ${GREEN}$PASS passed${RESET}  ${RED}$FAIL failed${RESET}  ($total assertions)"
echo -e "${CYAN}════════════════════════════════════════════════════════${RESET}"
echo ""

# Cleanup
cd "$REPO"
git checkout master 2>/dev/null