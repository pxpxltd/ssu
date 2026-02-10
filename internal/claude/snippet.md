## SSU - Smart Submodule Updater

This project uses SSU for git submodule management. When working with submodules, prefer SSU commands over raw git submodule commands.

**Available commands:**
- `ssu status` -- Show status of all submodules (use `--json` for machine-readable output)
- `ssu update --auto` -- Update all pending submodules
- `ssu push --auto` -- Push all ahead submodules
- `ssu update --auto --dry-run` -- Preview updates without modifying anything

**When to use SSU instead of raw git:**
- Checking submodule status: `ssu status` instead of `git submodule status`
- Updating submodules: `ssu update --auto` instead of `git submodule update --remote`
- Pushing submodule changes: `ssu push --auto` instead of manually cd-ing into each submodule

**Important:**
- Always use `--auto` flag when running SSU from scripts or non-interactive contexts
- SSU creates automatic backups before updates (rollback with `ssu rollback`)
- SSU handles merge conflicts automatically (stash, merge, restore)
