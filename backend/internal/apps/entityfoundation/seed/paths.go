package seed

import "path/filepath"

const DefaultSeedDir = "data/entity_foundation"

func DefaultSeedPaths(seedDir string) []string {
	if seedDir == "" {
		seedDir = DefaultSeedDir
	}
	names := []string{
		"alliance_orgs.json",
		"economies.json",
		"policy_bodies.json",
		"markets.json",
		"indices.json",
		"sectors.json",
		"chain_nodes.json",
		"metrics.json",
		"commodities.json",
		"companies.json",
		"securities.json",
		"instruments.json",
		"persons.json",
		"relationships.json",
	}
	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, filepath.Join(seedDir, name))
	}
	return paths
}
