package eventtagcatalog

import (
	"context"
	"testing"
)

type fakeRepository struct {
	tags []Tag
	err  error
}

func (r fakeRepository) ListActive(context.Context) ([]Tag, error) {
	return append([]Tag(nil), r.tags...), r.err
}

func TestServiceReturnsDeterministicActiveCatalog(t *testing.T) {
	service := NewService(fakeRepository{tags: []Tag{
		{ID: "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea", Kind: "news_category", Code: "technology_industry", Name: "科技产业", Active: true},
		{ID: "173cabde-c2bf-5cdc-a026-08cd52a953f0", Kind: "index_category", Code: "macro_economic_index", Name: "宏观经济指数", Active: true},
	}})

	catalog, err := service.Active(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tags) != 2 ||
		catalog.Tags[0].Kind != "index_category" ||
		catalog.Tags[1].Kind != "news_category" {
		t.Fatalf("catalog tags = %#v", catalog.Tags)
	}
	const wantHash = "1a7035195312cdf7652880308d9fffcc6aea180f7c09b5c07f6678514a1298eb"
	if catalog.Hash != wantHash || catalog.Revision != "event-tags:"+wantHash {
		t.Fatalf("catalog identity = %q %q", catalog.Revision, catalog.Hash)
	}
}

func TestServiceRejectsInactiveOrInvalidRepositoryRows(t *testing.T) {
	for _, test := range []struct {
		name string
		tag  Tag
	}{
		{name: "inactive", tag: Tag{ID: "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea", Kind: "news_category", Code: "technology_industry", Name: "科技产业"}},
		{name: "unknown kind", tag: Tag{ID: "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea", Kind: "other", Code: "technology_industry", Name: "科技产业", Active: true}},
		{name: "blank code", tag: Tag{ID: "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea", Kind: "news_category", Name: "科技产业", Active: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewService(fakeRepository{tags: []Tag{test.tag}}).Active(context.Background())
			if err == nil {
				t.Fatal("Active error = nil")
			}
		})
	}
}
