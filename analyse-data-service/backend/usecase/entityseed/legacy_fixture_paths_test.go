package seed

import "path/filepath"

func legacyFixturePaths(seedDir string) []string {
	paths := DefaultSeedPaths(seedDir)
	for _, name := range []string{
		"sectors.json",
		"sector_source_mappings.json",
		"chain_nodes.json",
		"industry_chains_v1.json",
		filepath.Join("relationships", "covers_sector.json"),
		filepath.Join("relationships", "tracked_by_benchmark.json"),
	} {
		paths = append(paths, filepath.Join(seedDir, name))
	}
	return paths
}
