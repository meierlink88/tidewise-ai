package biz

import (
	"context"
	"testing"
	"time"
)

func TestListSourcesFiltersAndPaginatesWhilePreservingDataOrder(t *testing.T) {
	updated := time.Date(2026, 8, 19, 2, 0, 0, 0, time.UTC)
	repo := &FakeDataServiceRepo{ListSourcesFunc: func(context.Context) ([]Source, error) {
		return []Source{
			{ID: "one", Code: "official-a", Name: "Official A", OwnershipType: "fixed", ChannelType: "api", Enabled: true, Priority: 1, DefaultSourceLevel: "L1_OFFICIAL", UpdatedAt: updated},
			{ID: "two", Code: "official-b", Name: "Official B", OwnershipType: "fixed", ChannelType: "api", Enabled: true, Priority: 1, DefaultSourceLevel: "L1_OFFICIAL", UpdatedAt: updated},
		}, nil
	}}
	enabled, priority := true, 1
	page, err := NewService(repo).ListSources(context.Background(), SourceListQuery{Text: "OFFICIAL", OwnershipType: "fixed", ChannelType: "api", Enabled: &enabled, Priority: &priority, DefaultSourceLevel: "L1_OFFICIAL", UpdatedFrom: &updated, UpdatedTo: &updated, Page: 2, PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || page.Page != 2 || len(page.Items) != 1 || page.Items[0].ID != "two" {
		t.Fatalf("page = %#v", page)
	}
}
