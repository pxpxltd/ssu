package engine

import (
	"testing"

	"github.com/pxpxltd/ssu/internal/git"
)

func TestNew(t *testing.T) {
	mock := &git.MockGitService{}
	eng := New(mock)
	if eng == nil {
		t.Fatal("New returned nil")
	}
	if eng.git == nil {
		t.Fatal("Engine.git is nil")
	}
}
