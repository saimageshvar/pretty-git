# pgit checkout — Implementation Summary

## What it does
Three-mode command that bridges native `git checkout` UX with pgit's TUI:

| Invocation                    | Behaviour                                              |
|-------------------------------|--------------------------------------------------------|
| `pgit checkout`               | Branch switcher UI (identical to `pgit branch`)        |
| `pgit checkout <name>`        | Switch directly if branch exists; open create form if not |
| `pgit checkout -b [name] ...` | Inline TUI form to create a new branch                 |

If all three fields (name, parent, desc) are supplied via flags, no TUI opens.

## Command routing (checkout.go)
```
runCheckout(args)
  ├── len(args)==0              → runBranch()
  ├── args[0] has no "-" prefix → BranchExists? SwitchBranch : runCheckoutCreate(["-b", name])
  └── args[0]=="-b"             → runCheckoutCreate(args)
```

## Key files
- `cmd/pretty-git/checkout.go` — routing (`runCheckout`), create form runner (`runCheckoutCreate`), `printCreated`
- `internal/ui/checkout/model.go` — Bubble Tea create form model
- `internal/git/git.go` — `CreateBranch`, `SwitchBranch`, `BranchExists` (added)

## TUI form — 3 fields, Tab/Shift+Tab to navigate
| Field    | Widget      | Notes                                  |
|----------|-------------|----------------------------------------|
| Branch   | textinput   | required; Enter validates + advances   |
| Parent   | textinput   | type-to-filter; picker opens below     |
| Desc     | textinput   | optional; Enter submits                |

## Parent picker (below layout only)
- Appears below form divider when Parent field is focused
- Branches in DFS tree order (`├─` / `└─`), same as `pgit branch`
- Description uses all remaining line width: `descW = width - indent - prefixLen - nameMax - 2`
- `↑/↓` navigate, `enter` selects & advances to Desc, `ctrl+d` clears, `tab` advances without selecting

## Focus auto-advance
Constructor skips pre-filled fields: name given → start on Parent; name+parent given → start on Desc.
`preselectParent()` scrolls picker to the already-selected branch when re-entering the field.

## Git ops sequence (async tea.Cmd)
`CreateBranch` → `SetParent` (if set) → `SetDescription` (if set) → `createDoneMsg`
Errors at any step surface in the footer; focus returns to Branch field.
