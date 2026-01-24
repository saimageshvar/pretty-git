Verification steps for `--update-parent` and backup behavior

1. Build the project

   go build ./...

2. Create a temporary repo to test (or use an existing test repo)

   git init repo-test && cd repo-test
   git commit --allow-empty -m "root" --no-edit
   git branch base
   git checkout -b feature

3. Test setting parent when creating a branch

   # from 'feature', create 'new-branch' and record parent
   pretty-git checkout -b new-branch
   # verify with git config --get pretty-git.parent.new-branch

4. Test updating parent on existing branch

   # switch back to 'feature' then
   pretty-git checkout base
   # switch to an existing branch and update parent
   pretty-git checkout feature --update-parent
   # if you are prompted, answer 'y' or use --yes to skip prompt
   # verify backup was created:
   git config --get pretty-git.parent.backup.feature
   # verify current parent:
   git config --get pretty-git.parent.feature

5. Undo (manual)

   # if needed, restore backup:
   git config --get pretty-git.parent.backup.feature
   git config --local pretty-git.parent.feature "<value-from-backup>"

Notes

- Backups are stored under `pretty-git.parent.backup.<branch>` in the repository local git config.
- `--update-parent` must be specified to overwrite an existing parent entry; without it the command will error if a parent exists.
- Use `--yes` (or `-y`) to skip interactive confirmation when overwriting an existing parent.
