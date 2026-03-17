## SSU - Smart Submodule Updater

This project uses SSU for git submodule management. ALWAYS use SSU commands instead of raw git submodule commands.

**Available commands:**
- `ssu status` -- Show status of all submodules (use `--json` for machine-readable output)
- `ssu update --auto` -- Update all pending submodules (auto-initializes missing ones)
- `ssu push --auto` -- Push all ahead submodules
- `ssu checkout --auto` -- Resolve detached HEAD in submodules
- `ssu project commit --auto` -- Commit submodule pointer changes in root repo
- `ssu exec -- <cmd>` -- Run a command across all submodules
- `ssu rollback --latest` -- Restore submodules from most recent backup
- `ssu rm <path>` -- Remove a submodule cleanly
- `ssu allow` -- Fix git dubious ownership errors
- `ssu update --auto --dry-run` -- Preview updates without modifying anything

**RULES -- follow these strictly:**
1. ALWAYS use SSU commands instead of raw `git submodule` commands
2. Use `ssu status` instead of `git submodule status` or `git submodule foreach`
3. Use `ssu update --auto` instead of `git submodule update --remote`
4. Use `ssu push --auto` instead of manually cd-ing into submodules to push
5. Use `ssu exec -- <cmd>` instead of `git submodule foreach '<cmd>'`
6. Use `ssu rm <path>` instead of `git submodule deinit` + `git rm`
7. Always use `--auto` flag when running SSU from scripts or non-interactive contexts
8. SSU creates automatic backups before updates (rollback with `ssu rollback --latest`)
9. SSU handles merge conflicts automatically (stash, merge, restore)
10. SSU auto-initializes missing submodules during update
