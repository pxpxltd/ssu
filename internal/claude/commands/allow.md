---
description: Fix git dubious ownership errors in submodules. Use this when git refuses to operate due to ownership mismatch.
allowed-tools:
  - Bash
---

Fix git "dubious ownership" errors in submodules using SSU.

When git refuses to operate on submodules with "fatal: detected dubious ownership in repository", use this command to add the directories to git's safe.directory list.

Fix all submodules:

```
ssu allow
```

Fix a specific submodule:

```
ssu allow <submodule-path>
```

This runs `git config --global --add safe.directory <path>` for each affected submodule.

$ARGUMENTS
