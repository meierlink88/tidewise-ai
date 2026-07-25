package data

import (
	"errors"
	"testing"
	"time"
)

func TestRawDocumentPageWireValidatesAndMaps(t *testing.T) {
	wire := rawDocumentPageWire{
		Items: []rawDocumentWire{{
			ID: "raw-1", ContractVersion: 1, SourceName: "BBC", Title: "Headline",
			IngestChannel: "rss", SourceType: "news", SourceURL: "https://example.com",
			ContentText: "body", ContentLevel: "full", RawObjectURI: "s3://bucket/raw-1",
			RawMIMEType: "text/plain", Language: "en", CollectedAt: time.Now(),
			IngestStatus: "collected", ContentSHA256: "abc",
		}},
		Page: 1, PageSize: 50, Total: 1,
	}
	got, err := wire.toBiz()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "raw-1" {
		t.Fatalf("mapped page = %#v", got)
	}
}

func TestEventPageWireRejectsUnknownStatus(t *testing.T) {
	_, err := (eventPageWire{
		Items: []eventWire{{
			ID: "event-1", Title: "Event", FirstSeenAt: time.Now(),
			EventStatus: "invented", FactStatus: "verified",
		}},
		Page: 1, PageSize: 50, Total: 1,
	}).toBiz()
	var decodeErr *Error
	if !errors.As(err, &decodeErr) || decodeErr.Kind != ErrorKindDecode {
		t.Fatalf("error = %v, want decode error", err)
	}
}
