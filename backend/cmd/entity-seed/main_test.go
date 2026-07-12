package main

import (
	"os"
	"strings"
	"testing"
)

func TestEntitySeedRequiresExplicitMutuallyExclusiveConvergenceFlags(t *testing.T) {
	content, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, fragment := range []string{"apply-sector-convergence", "apply-sector-convergence-correction", "cannot be used together"} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("main.go missing %q", fragment)
		}
	}
}
