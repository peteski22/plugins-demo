package pipeline

import (
	"bytes"
	"io"
	"net/http"

	pb "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1/plugins"
)

// Middleware returns a Chi-compatible middleware that processes requests through the plugin pipeline.
func (p *Pipeline) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 1. Convert http.Request → pb.HTTPRequest
			httpReq, err := httpRequestToProto(r)
			if err != nil {
				http.Error(w, "Failed to process request", http.StatusInternalServerError)
				p.logger.Error("failed to convert http request to proto", "error", err)
				return
			}

			// 2. Run REQUEST flow through pipeline
			httpResp, err := p.RunRequest(ctx, httpReq)
			if err != nil {
				// Pipeline error (required plugin failed, or infrastructure error)
				http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
				p.logger.Error("pipeline request flow failed", "error", err)
				return
			}

			// 3. Check if plugin short-circuited (business logic rejection)
			if !httpResp.Continue {
				// Plugin wants to respond directly (e.g., 429 rate limit, 403 auth failed)
				writeHTTPResponse(w, httpResp)
				return
			}

			// 4. Continue to actual handler (capture response)
			recorder := newResponseRecorder(w)
			next.ServeHTTP(recorder, r)

			// 5. Convert handler response → pb.HTTPResponse
			handlerResp := &pb.HTTPResponse{
				StatusCode: int32(recorder.statusCode),
				Headers:    convertHeadersToMap(recorder.Header()),
				Body:       recorder.body.Bytes(),
				Continue:   true,
			}

			// 6. Run RESPONSE flow through pipeline
			finalResp, err := p.RunResponse(ctx, handlerResp)
			if err != nil {
				// Pipeline error in response flow
				http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
				p.logger.Error("pipeline response flow failed", "error", err)
				return
			}

			// 7. Write final (potentially modified) response
			writeHTTPResponse(w, finalResp)
		})
	}
}

// httpRequestToProto converts *http.Request → *pb.HTTPRequest.
func httpRequestToProto(r *http.Request) (*pb.HTTPRequest, error) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body)) // Restore body for downstream handlers

	// Convert headers (take first value only)
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return &pb.HTTPRequest{
		Method:     r.Method,
		Url:        r.URL.String(),
		Path:       r.URL.Path,
		Headers:    headers,
		Body:       body,
		RemoteAddr: r.RemoteAddr,
		RequestUri: r.RequestURI,
	}, nil
}

// writeHTTPResponse writes *pb.HTTPResponse → http.ResponseWriter.
func writeHTTPResponse(w http.ResponseWriter, resp *pb.HTTPResponse) {
	// Write headers
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}

	// Write status code
	if resp.StatusCode > 0 {
		w.WriteHeader(int(resp.StatusCode))
	}

	// Write body
	if len(resp.Body) > 0 {
		_, _ = w.Write(resp.Body)
	}
}

// convertHeadersToMap converts http.Header → map[string]string (first value only).
func convertHeadersToMap(h http.Header) map[string]string {
	m := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

// responseRecorder captures the response from the next handler.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

// newResponseRecorder creates a new responseRecorder.
func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status
	}
}

// WriteHeader captures the status code.
func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Write captures the response body.
func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
