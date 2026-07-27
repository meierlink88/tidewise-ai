package industrygraphprojection

import (
	"strings"
	"testing"
)

func TestValidateFrozenV1ProjectionRejectsUnexpectedPackage(t *testing.T) {
	t.Parallel()

	err := ValidateFrozenV1Projection(validTestProjection())
	if err == nil || !strings.Contains(err.Error(), "frozen V1 package SHA-256") {
		t.Fatalf(
			"ValidateFrozenV1Projection() error = %v, want frozen package SHA rejection",
			err,
		)
	}
}

func TestValidateFrozenV1ProjectionRejectsCountDrift(t *testing.T) {
	t.Parallel()

	projection := validTestProjection()
	projection.PackageSHA256 = FrozenV1PackageSHA256

	err := ValidateFrozenV1Projection(projection)
	if err == nil || !strings.Contains(err.Error(), "node count 5, want frozen V1 count 4449") {
		t.Fatalf(
			"ValidateFrozenV1Projection() error = %v, want frozen node-count rejection",
			err,
		)
	}
}
