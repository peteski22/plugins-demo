package plugin

//
//// TODO: Surely these need to go...
//// Capability defines behavior constraints for a plugin grouping.
//// These properties are enforced by the pipeline during execution.
//type Capability struct {
//	// Mode determines if plugins execute sequentially or concurrently.
//	Mode ExecutionMode
//
//	// CanReject when true allows plugins to stop request processing and return errors.
//	CanReject bool
//
//	// CanModify when true allows plugins to mutate request/response data.
//	CanModify bool
//}
//
//// Predefined capabilities with their execution characteristics.
//var (
//	// AuthN verifies the identity of the requester.
//	AuthN = Capability{Mode: ExecSerial, CanReject: true}
//
//	// AuthZ determines if the authenticated entity has permission for the requested action.
//	AuthZ = Capability{Mode: ExecSerial, CanReject: true}
//
//	// RateLimiting enforces request rate limits to prevent abuse.
//	RateLimiting = Capability{Mode: ExecSerial, CanReject: true}
//
//	// Validation checks request structure, schema, or business rules.
//	Validation = Capability{Mode: ExecSerial, CanReject: true}
//
//	// Content transforms or enriches request/response bodies.
//	Content = Capability{Mode: ExecSerial, CanReject: true, CanModify: true}
//
//	// Observability provides metrics, traces, and logging without blocking requests.
//	Observability = Capability{Mode: ExecParallel, CanReject: false}
//)
