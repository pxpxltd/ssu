---
description: Update git submodules using SSU
allowed-tools:
  - Bash
---

Update git submodules using SSU in automatic mode.

IMPORTANT: Always use `--auto` mode. Claude Code does not have a TTY, so interactive selection is not available.

First, check current status:

```
ssu status
```

Then update all pending submodules:

```
ssu update --auto
```

For a dry-run preview without making changes:

```
ssu update --auto --dry-run
```

If the user specifies particular submodules or options, incorporate them via $ARGUMENTS.

$ARGUMENTS
