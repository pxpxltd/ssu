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

Report the results clearly, highlighting:
- Submodules that need updates (pending)
- Submodules with unpushed changes (ahead)
- Submodules with local modifications (modified)
- Submodules that are not initialized (missing)
- Submodules with conflicts

If there are errors, suggest the appropriate SSU command to fix them (e.g., `ssu update --auto` for pending, `ssu push --auto` for ahead, `ssu checkout --auto` for detached HEAD).

$ARGUMENTS
