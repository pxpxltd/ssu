---
description: Restore submodules from a backup after a failed update. Use this when an SSU update went wrong.
allowed-tools:
  - Bash
---

Restore submodules to a previous state using SSU's backup system.

SSU creates automatic backups before every update. Use rollback to restore from a backup if something went wrong.

List available backups:

```
ssu rollback --list
```

Rollback to the most recent backup:

```
ssu rollback --latest
```

Rollback to a specific backup file:

```
ssu rollback <backup-file>
```

Note: Rollback restores exact commit SHAs, which may leave submodules in detached HEAD state. Run `ssu checkout --auto` afterwards to resolve.

$ARGUMENTS
