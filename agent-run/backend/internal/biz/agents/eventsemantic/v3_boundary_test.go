package eventsemantic

import (
	"reflect"
	"testing"
)

func TestV3PortsAndSubmissionExposeNoImpactThemeOrTransmissionSurface(t *testing.T) {
	dataClient := reflect.TypeOf((*DataClient)(nil)).Elem()
	for _, method := range []string{
		"ListDirectTargets", "GetDirectTargets", "ListTransmissionRules", "GetTransmissionRules",
		"CreateTheme", "CreateReasonTree",
	} {
		if _, exists := dataClient.MethodByName(method); exists {
			t.Fatalf("Event Semantic V3 DataClient exposes prohibited method %s", method)
		}
	}
	submission := reflect.TypeOf(SubmissionRequest{})
	for _, field := range []string{"DirectImpacts", "DirectTargets", "TransmissionRules", "Themes", "ReasonTrees"} {
		if _, exists := submission.FieldByName(field); exists {
			t.Fatalf("Event Semantic V3 SubmissionRequest exposes prohibited field %s", field)
		}
	}
}
