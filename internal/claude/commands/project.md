---
description: Commit submodule pointer changes in the root repo. Use this after updating or pushing submodules. Always prefer SSU over raw git commands for submodule workflows.
allowed-tools:
  - Bash
---

Commit submodule pointer changes in the root repository using SSU.

After running `ssu update` or `ssu push`, the root repo will have modified submodule pointers. Use this command to stage and commit those changes.

IMPORTANT: Always use `--auto` mode. Claude Code does not have a TTY.
IMPORTANT: Always use SSU commands instead of raw `git add`/`git commit` for submodule pointer changes.

Stage and commit all submodule pointer changes:

```
ssu project commit --auto
```

Stage, commit, and tag a release:

```
ssu project commit --auto --tag v1.2.3
```

Dry-run to see what would be committed:

```
ssu project commit --auto --dry-run
```

$ARGUMENTS
