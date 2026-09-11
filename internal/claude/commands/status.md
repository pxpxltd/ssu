---
description: Check git submodule status. Use this INSTEAD of 'git submodule status' or 'git submodule foreach'. Always prefer SSU over raw git submodule commands.
allowed-tools:
  - Bash
---

Check the status of all git submodules in this project using SSU.

IMPORTANT: Always use SSU commands instead of raw `git submodule` commands. SSU provides richer output with branch detection, ahead/behind counts, and color-coded status.

Run:

```
ssu status
```

For machine-readable JSON output:

```
ssu status --json
```

To see open GitHub pull requests for the root repository and every submodule (with URLs), add `--pr`. This needs the GitHub CLI (`gh`) to be installed and logged in:

```
ssu status --pr
ssu status --pr --json
```

With `--pr --json`, each module that was looked up carries either a `pull_requests` array (number, title, url, head_branch, base_branch, author, draft) or a `pr_error` message; missing and skipped modules omit both. `pull_requests_truncated: true` means the repository has more than the 100 PRs listed.

Report the results clearly, highlighting:
- Submodules that need updates (pending)
- Submodules with unpushed changes (ahead)
- Submodules with local modifications (modified)
- Submodules that are not initialized (missing)
- Submodules with conflicts
- Open pull requests and their links (when `--pr` was used)

If there are errors, suggest the appropriate SSU command to fix them (e.g., `ssu update --auto` for pending, `ssu push --auto` for ahead, `ssu checkout --auto` for detached HEAD).

$ARGUMENTS
