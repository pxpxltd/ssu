---
description: Show git submodule status using SSU
allowed-tools:
  - Bash
---

Check the status of all git submodules in this project using SSU.

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
- Submodules with conflicts

$ARGUMENTS
