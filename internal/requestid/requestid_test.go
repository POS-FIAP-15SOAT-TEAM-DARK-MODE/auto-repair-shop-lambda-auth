package requestid_test

import (
	"testing"

	"github.com/POS-FIAP-15SOAT-TEAM-DARK-MODE/auto-repair-shop-lambda-auth/internal/requestid"
)

func TestFrom_PrefersIncomingHeader(t *testing.T) {
	got := requestid.From(map[string]string{"x-request-id": "upstream-request-id-123"}, "gateway-generated-id")

	if got != "upstream-request-id-123" {
		t.Fatalf("From() = %q, want the incoming header value", got)
	}
}

func TestFrom_HeaderLookupIsCaseInsensitive(t *testing.T) {
	got := requestid.From(map[string]string{"X-Request-Id": "mixed-case-header-id"}, "gateway-generated-id")

	if got != "mixed-case-header-id" {
		t.Fatalf("From() = %q, want the incoming header value regardless of case", got)
	}
}

func TestFrom_FallsBackToGatewayRequestID(t *testing.T) {
	got := requestid.From(map[string]string{}, "gateway-generated-id")

	if got != "gateway-generated-id" {
		t.Fatalf("From() = %q, want the gateway's own request ID", got)
	}
}

func TestFrom_IgnoresEmptyHeaderValue(t *testing.T) {
	got := requestid.From(map[string]string{"x-request-id": ""}, "gateway-generated-id")

	if got != "gateway-generated-id" {
		t.Fatalf("From() = %q, want the fallback when the header is present but empty", got)
	}
}
