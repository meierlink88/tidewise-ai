package eventtagcatalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type Tag struct {
	ID     string `json:"id"`
	Kind   string `json:"tag_kind"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Active bool   `json:"is_active"`
}

type Catalog struct {
	Revision string
	Hash     string
	Tags     []Tag
}

type Repository interface {
	ListActive(context.Context) ([]Tag, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Active(ctx context.Context) (Catalog, error) {
	if s == nil || s.repository == nil {
		return Catalog{}, fmt.Errorf("Event Tag Catalog repository is required")
	}
	tags, err := s.repository.ListActive(ctx)
	if err != nil {
		return Catalog{}, fmt.Errorf("list active Event Tags: %w", err)
	}
	if len(tags) == 0 {
		return Catalog{}, fmt.Errorf("active Event Tag Catalog is empty")
	}
	for position := range tags {
		tags[position].ID = strings.TrimSpace(tags[position].ID)
		tags[position].Kind = strings.TrimSpace(tags[position].Kind)
		tags[position].Code = strings.TrimSpace(tags[position].Code)
		tags[position].Name = strings.TrimSpace(tags[position].Name)
		if err := validateTag(tags[position]); err != nil {
			return Catalog{}, fmt.Errorf("invalid Event Tag Catalog row: %w", err)
		}
	}
	sort.Slice(tags, func(left, right int) bool {
		if tags[left].Kind != tags[right].Kind {
			return tags[left].Kind < tags[right].Kind
		}
		if tags[left].Code != tags[right].Code {
			return tags[left].Code < tags[right].Code
		}
		return tags[left].ID < tags[right].ID
	})
	for position := 1; position < len(tags); position++ {
		if tags[position-1].ID == tags[position].ID ||
			(tags[position-1].Kind == tags[position].Kind && tags[position-1].Code == tags[position].Code) {
			return Catalog{}, fmt.Errorf("Event Tag Catalog contains duplicate identity")
		}
	}
	encoded, err := json.Marshal(tags)
	if err != nil {
		return Catalog{}, fmt.Errorf("encode Event Tag Catalog: %w", err)
	}
	digest := sha256.Sum256(encoded)
	hash := hex.EncodeToString(digest[:])
	return Catalog{
		Revision: "event-tags:" + hash,
		Hash:     hash,
		Tags:     tags,
	}, nil
}

func validateTag(tag Tag) error {
	if _, err := uuid.Parse(tag.ID); err != nil {
		return fmt.Errorf("Tag ID is invalid")
	}
	if tag.Kind != "news_category" && tag.Kind != "index_category" {
		return fmt.Errorf("Tag kind is invalid")
	}
	if tag.Code == "" || tag.Name == "" {
		return fmt.Errorf("Tag code and name are required")
	}
	if !tag.Active {
		return fmt.Errorf("Tag must be active")
	}
	return nil
}
