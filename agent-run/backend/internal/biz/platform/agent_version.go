package agentrun

import (
	"context"
	"errors"
	"strings"
)

var ErrAgentVersionPublicationInvalid = errors.New("Agent Version publication is invalid")

type AgentVersionPublicationStore interface {
	PublishAgentVersions(context.Context, []AgentVersion) ([]AgentVersion, error)
	WithdrawAgentVersions(context.Context, []AgentVersion) error
}

type AgentVersionPublication struct {
	Added []AgentVersion
}

type AgentVersionPublisher struct {
	store AgentVersionPublicationStore
}

func NewAgentVersionPublisher(store AgentVersionPublicationStore) (*AgentVersionPublisher, error) {
	if store == nil {
		return nil, errors.New("Agent Version Publication Store is required")
	}
	return &AgentVersionPublisher{store: store}, nil
}

func (p *AgentVersionPublisher) PublishCurrent(
	ctx context.Context,
	versions []AgentVersion,
) (AgentVersionPublication, error) {
	validated, err := validateAgentVersions(versions)
	if err != nil {
		return AgentVersionPublication{}, err
	}
	added, err := p.store.PublishAgentVersions(ctx, validated)
	if err != nil {
		return AgentVersionPublication{}, err
	}
	return AgentVersionPublication{Added: added}, nil
}

func (p *AgentVersionPublisher) Withdraw(
	ctx context.Context,
	publication AgentVersionPublication,
) error {
	if len(publication.Added) == 0 {
		return nil
	}
	validated, err := validateAgentVersions(publication.Added)
	if err != nil {
		return err
	}
	return p.store.WithdrawAgentVersions(ctx, validated)
}

func validateAgentVersions(versions []AgentVersion) ([]AgentVersion, error) {
	if len(versions) == 0 {
		return nil, ErrAgentVersionPublicationInvalid
	}
	seenVersions := make(map[string]string, len(versions))
	validated := make([]AgentVersion, 0, len(versions))
	for _, candidate := range versions {
		if candidate.AgentKey == "" || candidate.Version == "" ||
			candidate.AgentKey != strings.TrimSpace(candidate.AgentKey) ||
			candidate.Version != strings.TrimSpace(candidate.Version) {
			return nil, ErrAgentVersionPublicationInvalid
		}
		if agentKey, exists := seenVersions[candidate.Version]; exists {
			if agentKey != candidate.AgentKey {
				return nil, ErrAgentVersionPublicationInvalid
			}
			continue
		}
		seenVersions[candidate.Version] = candidate.AgentKey
		validated = append(validated, candidate)
	}
	return validated, nil
}
