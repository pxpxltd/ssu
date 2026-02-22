package engine

// ProgressEventType identifies the kind of progress event.
type ProgressEventType int

const (
	// EventStarted fires when processing begins for a submodule.
	EventStarted ProgressEventType = iota
	// EventCompleted fires when processing finishes successfully for a submodule.
	EventCompleted
	// EventFailed fires when processing fails for a submodule.
	EventFailed
)

// ProgressEvent carries information about scan/update/push progress.
type ProgressEvent struct {
	Type   ProgressEventType // Started, Completed, or Failed
	Path   string            // Submodule path (or "." for root)
	Phase  string            // "fetch", "status", "update", "push"
	Error  error             // Non-nil for EventFailed
	Total  int               // Total number of items being processed
	Done   int               // Number of items completed so far
	Action string            // Human-readable result (e.g. "pushed", "merged 3 commits")
}

// ProgressFunc is a callback that receives progress events during operations.
// Implementations must be safe for concurrent calls from multiple goroutines.
type ProgressFunc func(ProgressEvent)
