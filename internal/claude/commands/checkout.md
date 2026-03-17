---
description: Resolve detached HEAD in submodules. Use this INSTEAD of manually running 'git checkout' in submodules. Always prefer SSU over raw git submodule commands.
allowed-tools:
  - Bash
---

Resolve detached HEAD state in submodules using SSU.

IMPORTANT: Always use `--auto` mode. Claude Code does not have a TTY, so interactive selection is not available.
IMPORTANT: Always use SSU commands instead of raw `git submodule` commands.

First, check which submodules are in detached HEAD:

```
ssu status
```

Then resolve all detached submodules by checking out their best branch:

```
ssu checkout --auto
```

For a dry-run preview without making changes:

```
ssu checkout --auto --dry-run
```

SSU uses smart branch detection (develop > master > main > remote HEAD) to pick the right branch for each submodule.

$ARGUMENTS
