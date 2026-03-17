---
description: Run a command across all submodules. Use this INSTEAD of 'git submodule foreach'. Always prefer SSU over raw git submodule commands.
allowed-tools:
  - Bash
---

Run an arbitrary command in every submodule using SSU.

IMPORTANT: Always use SSU commands instead of raw `git submodule foreach`.

Run a command across all submodules:

```
ssu exec -- git status
```

```
ssu exec -- git stash list
```

```
ssu exec -- ls -la
```

The `--` separator is required before the command. SSU runs the command in each submodule directory and reports results with the submodule path prefixed.

$ARGUMENTS
