package client

import (
	"errors"

	"github.com/redpanda-data/common-go/rpadmin"
	"github.com/twmb/franz-go/pkg/kerr"
)

// For a list of errors from the Kafka API see:
//
// https://github.com/twmb/franz-go/blob/b77dd13e2bfaee7f5181df27b40ee4a4f6a73b09/pkg/kerr/kerr.go#L76-L192
var terminalClientErrors = []error{
	kerr.UnsupportedSaslMechanism, kerr.InvalidRequest, kerr.PolicyViolation,
	kerr.SecurityDisabled, kerr.SaslAuthenticationFailed, kerr.InvalidPrincipalType,
}

// IsTerminalClientError returns whether or not the error comes
// from a terminal error from a failed API request by one of our clients.
func IsTerminalClientError(err error) bool {
	for _, terminal := range terminalClientErrors {
		if errors.Is(err, terminal) {
			return true
		}
	}

	// For our REST API we check to see if we have a 400 range
	// response, which shouldn't be retried.
	var restError *rpadmin.HTTPResponseError
	if errors.As(err, &restError) {
		code := restError.Response.StatusCode
		if code >= 400 && code < 500 {
			// we have a terminal error
			return true
		}
	}

	return false
}
