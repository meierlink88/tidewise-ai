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
		"benchmarks.json",
		"sectors.json",
		"chain_nodes.json",
		"metrics.json",
		"commodities.json",
		"companies.json",
		"securities.json",
		"instruments.json",
		"persons.json",
		filepath.Join("relationships", "member_of.json"),
		filepath.Join("relationships", "has_market.json"),
		filepath.Join("relationships", "tracks_index.json"),
		filepath.Join("relationships", "observes_benchmark.json"),
		filepath.Join("relationships", "measures.json"),
		filepath.Join("relationships", "references.json"),
		filepath.Join("relationships", "issues.json"),
		filepath.Join("relationships", "participates_in.json"),
		filepath.Join("relationships", "affiliated_with.json"),
		filepath.Join("relationships", "applies_to.json"),
	}
	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, filepath.Join(seedDir, name))
	}
	return paths
}
