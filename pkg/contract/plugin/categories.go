package plugin

const (
	// CategoryAuthN verifies the identity of the requester.
	CategoryAuthN Category = "authentication"

	// CategoryAuthZ determines if the authenticated entity has permission for the requested action.
	CategoryAuthZ Category = "authorization"

	// CategoryRateLimiting enforces request rate limits to prevent abuse.
	CategoryRateLimiting Category = "rate-limiting"

	// CategoryValidation checks request structure, schema, or business rules.
	CategoryValidation Category = "validation"

	// CategoryContent transforms or enriches request/response bodies.
	CategoryContent Category = "content"

	// CategoryObservability provides metrics, traces, and logging without blocking requests.
	CategoryObservability Category = "observability"
)

type Category string

// CategoryProperties represents execution semantics for each Category.
type CategoryProperties struct {
	// Mode determines if plugins execute sequentially or concurrently.
	Mode ExecutionMode

	// CanReject when true allows category-level failures to cause pipeline failure.
	CanReject bool

	// CanModify when true allows plugins to mutate the request/response object.
	CanModify bool // TODO: This should be tied to the execution mode (to prevent threading issues changing the req).
}
