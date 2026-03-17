---
description: Remove a git submodule cleanly. Use this INSTEAD of manual 'git submodule deinit' + 'git rm'. Always prefer SSU over raw git submodule commands.
allowed-tools:
  - Bash
---

Remove a git submodule cleanly using SSU.

IMPORTANT: Always use SSU commands instead of manually running `git submodule deinit`, `git rm`, and editing `.gitmodules`.

Remove a submodule:

```
ssu rm <submodule-path>
```

For example:

```
ssu rm plugins/deprecated-module
```

SSU handles all cleanup steps: deinit, git rm, and .git/modules cleanup. The change is staged but not committed -- commit it yourself or use `ssu project commit --auto`.

$ARGUMENTS
