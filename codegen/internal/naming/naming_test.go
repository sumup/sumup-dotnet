package naming

import "testing"

func TestPascalIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"createCheckout", "CreateCheckout"},
		{"get_payment_methods", "GetPaymentMethods"},
		{"merchant", "Merchant"},
		{"APIClient", "ApiClient"},
		{"checkout-reference", "CheckoutReference"},
	}

	for _, tt := range tests {
		if got := PascalIdentifier(tt.input); got != tt.want {
			t.Fatalf("PascalIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
