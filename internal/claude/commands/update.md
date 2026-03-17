---
description: Update git submodules to latest. Use this INSTEAD of 'git submodule update --remote' or 'git pull' in submodules. Always prefer SSU over raw git submodule commands.
allowed-tools:
  - Bash
---

Update git submodules using SSU in automatic mode.

IMPORTANT: Always use `--auto` mode. Claude Code does not have a TTY, so interactive selection is not available.
IMPORTANT: Always use SSU commands instead of raw `git submodule` commands. SSU handles branch detection, conflict resolution, backups, and auto-initialization of missing submodules.

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

SSU will automatically initialize any missing (unregistered) submodules before updating. If conflicts occur, SSU will attempt automatic resolution via stash/merge/pop. Failed updates can be rolled back with `ssu rollback`.

If the user specifies particular submodules or options, incorporate them via $ARGUMENTS.

$ARGUMENTS
