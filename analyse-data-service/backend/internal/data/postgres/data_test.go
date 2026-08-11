package postgres

import "testing"

func TestResearchDictionariesRejectMalformedPersistedEntityTypeDefinition(t *testing.T) {
	definitionPrefix := `"type_key":"product","version":1,"name_zh":"产品","name_en":"Product","business_definition":"A marketable product","inclusion_criteria":["identifiable product"],"exclusion_criteria":["company"],"event_link_allowed":true,"signal_subject_allowed":true,`
	definitionSuffix := `"allowed_event_roles":["event_subject"],"status":"active"`
	for name, payload := range map[string]string{
		"invalid controlled value": `{"entity_type_definitions":[{` + definitionPrefix + `"direct_target_mode":"invalid",` + definitionSuffix + `}]}`,
		"unknown persisted field":  `{"entity_type_definitions":[{` + definitionPrefix + `"direct_target_mode":"allow",` + definitionSuffix + `,"unexpected":true}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeResearchDictionaries([]byte(payload)); err == nil {
				t.Fatal("decodeResearchDictionaries() error = nil, want malformed persisted Entity Type Definition rejection")
			}
		})
	}
}
