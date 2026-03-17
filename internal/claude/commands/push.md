---
description: Push ahead submodules. Use this INSTEAD of manually cd-ing into submodules to 'git push'. Always prefer SSU over raw git submodule commands.
allowed-tools:
  - Bash
---

Push submodules that have unpushed commits using SSU in automatic mode.

IMPORTANT: Always use `--auto` mode. Claude Code does not have a TTY, so interactive selection is not available.
IMPORTANT: Always use SSU commands instead of raw `git submodule` commands.

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

SSU automatically sets up tracking branches if needed. Submodules in detached HEAD state are skipped with a warning.

$ARGUMENTS
