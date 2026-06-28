---
phase: quick
plan: 005
type: execute
wave: 1
depends_on: []
autonomous: true
---

<objective>
Add versioned `ssu export <file>.json` and `ssu import <file>.json` commands so developers can share and safely reproduce selected submodule branch/SHA state.
</objective>

<tasks>

<task type="auto">
  <name>Implement stack persistence and synchronization</name>
  <action>
Add a validated, deterministic JSON stack format and a testable service that exports initialized submodules and imports selected modules. Import must fetch first, skip dirty or divergent targets, safely fast-forward exact commits, and fall back to origin branch tips when an exported SHA is unavailable.
  </action>
</task>

<task type="auto">
  <name>Wire commands, picker, reporting, and documentation</name>
  <action>
Register export/import Cobra commands, reuse the preselected multi-module picker, support auto/non-TTY and dry-run modes, report every skipped/fallback/failed category, and document the workflow.
  </action>
</task>

<task type="auto">
  <name>Verify safety and behavior</name>
  <action>
Add format, service, Git integration, and command registration tests. Run formatting, vet, build, and the complete test suite.
  </action>
</task>

</tasks>
