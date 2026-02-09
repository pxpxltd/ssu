package engine

import "github.com/pxpxltd/ssu/internal/git"

// Engine orchestrates scan, update, and push workflows.
// It operates on git repositories through the GitService interface,
// enabling full testability with MockGitService.
type Engine struct {
	git git.GitService
}

// New creates an Engine backed by the given GitService.
func New(svc git.GitService) *Engine {
	return &Engine{git: svc}
}
