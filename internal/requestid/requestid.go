// Package requestid resolves the ID used to correlate this lambda's log
// lines with the rest of the request's path through the system (API Gateway,
// and downstream, the main app).
package requestid

import "strings"

// From prefers a caller-supplied X-Request-Id (so a trace started upstream,
// e.g. by auto-repair-shop, survives the hop through this lambda) and falls
// back to API Gateway's own per-invocation request ID otherwise.
func From(headers map[string]string, gatewayRequestID string) string {
	for k, v := range headers {
		if strings.EqualFold(k, "X-Request-Id") && v != "" {
			return v
		}
	}
	return gatewayRequestID
}
