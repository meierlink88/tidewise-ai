package eventimport

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventimport"
)

func TestLoadPackagesDirectoryIsSortedAndStrict(t *testing.T) {
	dir := t.TempDir()
	first := validPackage()
	first.PackageID = "package-a"
	first.Review.PackageID = first.PackageID
	second := validPackage()
	second.PackageID = "package-b"
	second.Review.PackageID = second.PackageID
	for name, pkg := range map[string]any{"02.json": second, "01.json": first} {
		content, err := EncodeReport(pkg)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	packages, err := LoadPackages(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := packages[0].PackageID; got != "package-a" {
		t.Fatalf("first package = %q", got)
	}
}

func TestDryRunDoesNotRequireStore(t *testing.T) {
	report, err := DryRun(context.Background(), []domainimport.Package{validPackage()})
	if err != nil {
		t.Fatal(err)
	}
	if len(report) != 1 || report[0].Counts.Events != 1 || report[0].EventID == "" {
		t.Fatalf("unexpected report: %+v", report)
	}
}
