package plugin

const (
	// ExecSerial executes plugins sequentially in configuration order.
	ExecSerial ExecutionMode = iota // TODO: use strings (for output)

	// ExecParallel executes plugins concurrently via goroutines.
	ExecParallel
)

// ExecutionMode controls how the host executes plugins within a category:
// serial (ordered, one-at-a-time) or parallel (concurrent).
type ExecutionMode int
