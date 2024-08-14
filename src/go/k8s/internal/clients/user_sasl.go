package clients

import (
	"strings"

	"github.com/twmb/franz-go/pkg/kadm"
)

var (
	supportedSASLMechanisms = map[string]kadm.ScramMechanism{
		"SCRAM-SHA-256": kadm.ScramSha256,
		"SCRAM-SHA-512": kadm.ScramSha512,
	}
)

func normalizeSASL(mechanism string) (kadm.ScramMechanism, error) {
	sasl, ok := supportedSASLMechanisms[strings.ToUpper(mechanism)]
	if !ok {
		return 0, ErrUnsupportedSASLMechanism
	}

	return sasl, nil
}
