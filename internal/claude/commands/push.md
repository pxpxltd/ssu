---
description: Push ahead submodules using SSU
allowed-tools:
  - Bash
---

Push submodules that have unpushed commits using SSU in automatic mode.

IMPORTANT: Always use `--auto` mode. Claude Code does not have a TTY, so interactive selection is not available.

First, check which submodules are ahead:

```
ssu status
```

Then push all ahead submodules:

```
ssu push --auto
```

For a dry-run preview without pushing:

```
ssu push --auto --dry-run
```

$ARGUMENTS
