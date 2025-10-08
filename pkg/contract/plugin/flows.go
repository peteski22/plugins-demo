package plugin

const (
	// FlowRequest processes inbound requests before they reach the application.
	FlowRequest Flow = iota

	// FlowResponse processes outbound responses after the application responds.
	FlowResponse
)

// Flow indicates which request/response lifecycle point a plugin participates in.
type Flow int
