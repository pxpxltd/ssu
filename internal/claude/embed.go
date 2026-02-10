package claude

import "embed"

// CommandsFS contains the slash command markdown files for Claude Code.
//
//go:embed commands/*.md
var CommandsFS embed.FS

// SnippetContent contains the CLAUDE.md snippet text that teaches
// Claude Code about SSU's capabilities.
//
//go:embed snippet.md
var SnippetContent string
