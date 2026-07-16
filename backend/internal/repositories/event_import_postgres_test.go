package repositories

import (
	"reflect"
	"strings"
	"testing"
)

func TestDecodeReceiptUUIDJSONArray(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		want []string
	}{
		{"raw documents", `["11111111-1111-1111-1111-111111111111"]`, []string{"11111111-1111-1111-1111-111111111111"}},
		{"event sources", `["22222222-2222-2222-2222-222222222222","33333333-3333-3333-3333-333333333333"]`, []string{"22222222-2222-2222-2222-222222222222", "33333333-3333-3333-3333-333333333333"}},
		{"event tag maps", `["44444444-4444-4444-4444-444444444444","55555555-5555-5555-5555-555555555555","66666666-6666-6666-6666-666666666666"]`, []string{"44444444-4444-4444-4444-444444444444", "55555555-5555-5555-5555-555555555555", "66666666-6666-6666-6666-666666666666"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeReceiptUUIDJSONArray(tc.name, []byte(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("decoded IDs = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestDecodeReceiptUUIDJSONArrayRejectsEmptyAndMalformedValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
	}{
		{"empty", `[]`},
		{"malformed JSON", `not-json`},
		{"wrong JSON type", `{"id":"11111111-1111-1111-1111-111111111111"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeReceiptUUIDJSONArray("raw_document_ids", []byte(tc.data))
			if err == nil || !strings.Contains(err.Error(), "raw_document_ids") {
				t.Fatalf("decode error = %v, want raw_document_ids context", err)
			}
		})
	}
}
