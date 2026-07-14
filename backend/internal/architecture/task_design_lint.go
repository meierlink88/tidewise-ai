package architecture

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type taskDesignDiagnostic struct {
	Code    string
	Path    string
	Section string
	Field   string
	Message string
}

type taskDesignLintResult struct {
	Errors   []taskDesignDiagnostic
	Warnings []taskDesignDiagnostic
}

type taskDesignSection struct {
	Name    string
	Content string
}

type taskDesignTable struct {
	Header []string
	Rows   [][]string
}

type taskDesignGate struct {
	Package int
	Values  []string
}

type taskDesignArtifact struct {
	Path       string
	Sections   []taskDesignSection
	Gates      []taskDesignGate
	Budget     map[string]string
	BudgetRows [][]string
	Layers     [][]string
	Packages   []int
	PackageIDs map[int]bool
}

var taskDesignNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var taskDesignPackageHeadingPattern = regexp.MustCompile(`^([1-9][0-9]*)\. .+ Package$`)
var taskDesignUnsignedPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

var taskDesignGateHeader = []string{"Package", "Gate", "Risk", "Human", "Reason Code", "Allowed Scope"}
var taskDesignBudgetKeys = []string{"human_gates", "stateful_layers", "checkpoints", "full_test_runs", "continuous_automation_scope"}
var taskDesignLayerHeader = []string{"Layer", "Package", "Environment", "Order", "Scope", "Exclusions", "Recovery Evidence", "Recovery Baseline", "Expected Counts/Hash/Schema", "Before Assertions", "After Assertions", "Stop Conditions"}

func lintTaskDesignArtifacts(proposalPath, proposal, tasksPath, tasks string) taskDesignLintResult {
	result := taskDesignLintResult{}
	proposalArtifact := parseTaskDesignArtifact(proposalPath, proposal, true, &result)
	tasksArtifact := parseTaskDesignArtifact(tasksPath, tasks, false, &result)
	compareTaskDesignArtifacts(proposalArtifact, tasksArtifact, &result)
	lintTaskDesignMicroPackages(tasksArtifact, &result)
	return result
}

func lintTaskDesignRepository(root, explicit string) taskDesignLintResult {
	if explicit != "" {
		if !taskDesignNamePattern.MatchString(explicit) || explicit == "archive" {
			return taskDesignLintResult{Errors: []taskDesignDiagnostic{taskDesignError("explicit-change", explicit, "scope", "change", "explicit change must be one active kebab-case name")}}
		}
		changeDir := filepath.Join(root, "openspec", "changes", explicit)
		if info, err := os.Stat(changeDir); err != nil || !info.IsDir() {
			return taskDesignLintResult{Errors: []taskDesignDiagnostic{taskDesignError("explicit-change", explicit, "scope", "change", "explicit change is unknown or not active")}}
		}
		return lintTaskDesignChange(changeDir)
	}

	baseline, baselineResult := readTaskDesignBaseline(root)
	if len(baselineResult.Errors) != 0 {
		return baselineResult
	}
	result := baselineResult
	changesDir := filepath.Join(root, "openspec", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		result.Errors = append(result.Errors, taskDesignError("active-scan", changesDir, "scope", "changes", err.Error()))
		return result
	}
	active := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "archive" {
			active[entry.Name()] = filepath.Join(changesDir, entry.Name())
		}
	}
	archived := readTaskDesignArchivedNames(filepath.Join(changesDir, "archive"))
	for name, rows := range baseline {
		if len(rows) > 1 {
			result.Warnings = append(result.Warnings, taskDesignWarning("baseline-duplicate", baselinePath(root), "baseline", "change_name", fmt.Sprintf("%s appears %d times; only the first valid active row can skip", name, len(rows))))
		}
		if _, ok := active[name]; ok {
			result.Warnings = append(result.Warnings, taskDesignWarning("baseline-skip", baselinePath(root), "baseline", name, fmt.Sprintf("skipping active legacy change %s: %s; remove after explicit adoption", name, rows[0])))
			delete(active, name)
			continue
		}
		if archived[name] {
			result.Warnings = append(result.Warnings, taskDesignWarning("baseline-archived", baselinePath(root), "baseline", name, "baseline entry is archived and provides no skip capability"))
		} else {
			result.Warnings = append(result.Warnings, taskDesignWarning("baseline-unknown", baselinePath(root), "baseline", name, "baseline entry has no active or archived change directory"))
		}
	}
	names := make([]string, 0, len(active))
	for name := range active {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		result.merge(lintTaskDesignChange(active[name]))
	}
	return result
}

func lintTaskDesignChange(changeDir string) taskDesignLintResult {
	result := taskDesignLintResult{}
	proposalPath := filepath.Join(changeDir, "proposal.md")
	tasksPath := filepath.Join(changeDir, "tasks.md")
	proposal, err := os.ReadFile(proposalPath)
	if err != nil {
		result.Errors = append(result.Errors, taskDesignError("artifact-read", proposalPath, "artifact", "proposal", err.Error()))
	}
	tasks, tasksErr := os.ReadFile(tasksPath)
	if tasksErr != nil {
		result.Errors = append(result.Errors, taskDesignError("artifact-read", tasksPath, "artifact", "tasks", tasksErr.Error()))
	}
	if err != nil || tasksErr != nil {
		return result
	}
	return lintTaskDesignArtifacts(proposalPath, string(proposal), tasksPath, string(tasks))
}

func parseTaskDesignArtifact(path, content string, proposal bool, result *taskDesignLintResult) taskDesignArtifact {
	artifact := taskDesignArtifact{Path: path, Sections: splitTaskDesignSections(content), Budget: map[string]string{}, PackageIDs: map[int]bool{}}
	gateIndex := taskDesignSectionIndex(artifact.Sections, "Gate Map")
	budgetIndex := taskDesignSectionIndex(artifact.Sections, "Complexity Budget")
	if proposal {
		whyIndex := taskDesignSectionIndex(artifact.Sections, "Why")
		if whyIndex < 0 || gateIndex != whyIndex+1 {
			result.Errors = append(result.Errors, taskDesignError("gate-heading", path, "Gate Map", "heading", "Gate Map must be the first H2 after Why"))
		}
	} else if gateIndex != 0 {
		result.Errors = append(result.Errors, taskDesignError("gate-heading", path, "Gate Map", "heading", "Gate Map must be the first H2"))
	}
	if gateIndex < 0 {
		result.Errors = append(result.Errors, taskDesignError("gate-missing", path, "Gate Map", "heading", "missing Gate Map"))
	} else {
		table, ok := parseTaskDesignTable(artifact.Sections[gateIndex].Content)
		if !ok || !equalTaskDesignStrings(table.Header, taskDesignGateHeader) {
			result.Errors = append(result.Errors, taskDesignError("gate-header", path, "Gate Map", "columns", "expected fixed Gate Map columns"))
		} else {
			artifact.Gates = lintTaskDesignGates(path, table.Rows, result)
		}
	}
	if budgetIndex != gateIndex+1 || budgetIndex < 0 {
		result.Errors = append(result.Errors, taskDesignError("budget-heading", path, "Complexity Budget", "heading", "Complexity Budget must immediately follow Gate Map"))
	} else {
		table, ok := parseTaskDesignTable(artifact.Sections[budgetIndex].Content)
		if !ok || !equalTaskDesignStrings(table.Header, []string{"Key", "Value"}) {
			result.Errors = append(result.Errors, taskDesignError("budget-header", path, "Complexity Budget", "columns", "expected Key and Value columns"))
		} else {
			artifact.BudgetRows = table.Rows
			artifact.Budget = lintTaskDesignBudget(path, table.Rows, artifact.Gates, result)
		}
	}
	for _, section := range artifact.Sections {
		match := taskDesignPackageHeadingPattern.FindStringSubmatch(section.Name)
		if match == nil {
			continue
		}
		id, _ := strconv.Atoi(match[1])
		artifact.Packages = append(artifact.Packages, id)
		artifact.PackageIDs[id] = true
	}
	if !proposal {
		lintTaskDesignPackageMapping(path, artifact.Gates, artifact.Packages, result)
	}
	layerIndex := taskDesignSectionIndex(artifact.Sections, "Stateful Layer Map")
	statefulCount, _ := strconv.Atoi(artifact.Budget["stateful_layers"])
	if statefulCount > 0 && layerIndex < 0 {
		result.Errors = append(result.Errors, taskDesignError("stateful-map-missing", path, "Stateful Layer Map", "heading", "stateful_layers is greater than zero"))
	} else if layerIndex >= 0 {
		if statefulCount > 0 && layerIndex != budgetIndex+1 {
			result.Errors = append(result.Errors, taskDesignError("stateful-heading", path, "Stateful Layer Map", "heading", "Stateful Layer Map must immediately follow Complexity Budget"))
		}
		table, ok := parseTaskDesignTable(artifact.Sections[layerIndex].Content)
		if !ok || !equalTaskDesignStrings(table.Header, taskDesignLayerHeader) {
			result.Errors = append(result.Errors, taskDesignError("stateful-header", path, "Stateful Layer Map", "columns", "expected fixed Stateful Layer Map columns"))
		} else {
			artifact.Layers = table.Rows
			lintTaskDesignLayers(path, table.Rows, statefulCount, artifact.Gates, result)
		}
	}
	if statefulCount == 0 && len(artifact.Layers) != 0 {
		result.Errors = append(result.Errors, taskDesignError("stateful-count", path, "Stateful Layer Map", "rows", "stateful_layers=0 requires no data rows"))
	}
	return artifact
}

func lintTaskDesignGates(path string, rows [][]string, result *taskDesignLintResult) []taskDesignGate {
	validReasons := map[string]bool{"SPEC_SEMANTICS": true, "R3_OPERATION": true, "NEO4J": true, "SHARED_ENV": true, "DEPLOYMENT_SECURITY": true, "DRIFT_RECOVERY": true, "APPLY_FINAL": true, "GIT_COMPLETION": true}
	seen := map[int]bool{}
	gates := make([]taskDesignGate, 0, len(rows))
	for index, row := range rows {
		if len(row) != len(taskDesignGateHeader) {
			result.Errors = append(result.Errors, taskDesignError("gate-row", path, "Gate Map", "row", fmt.Sprintf("row %d has %d columns", index+1, len(row))))
			continue
		}
		packageID, err := strconv.Atoi(row[0])
		if err != nil || packageID < 1 || strconv.Itoa(packageID) != row[0] || seen[packageID] {
			result.Errors = append(result.Errors, taskDesignError("gate-package", path, "Gate Map", "Package", fmt.Sprintf("invalid or duplicate package %q", row[0])))
			continue
		}
		seen[packageID] = true
		if row[1] == "" || row[5] == "" {
			result.Errors = append(result.Errors, taskDesignError("gate-value", path, "Gate Map", "Gate/Allowed Scope", "values must be non-empty"))
		}
		if row[2] != "R0" && row[2] != "R1" && row[2] != "R2" && row[2] != "R3" {
			result.Errors = append(result.Errors, taskDesignError("gate-risk", path, "Gate Map", "Risk", fmt.Sprintf("invalid risk %q", row[2])))
		}
		if row[3] != "yes" && row[3] != "no" {
			result.Errors = append(result.Errors, taskDesignError("gate-human", path, "Gate Map", "Human", fmt.Sprintf("invalid Human %q", row[3])))
		} else if row[3] == "yes" && !validReasons[row[4]] {
			result.Errors = append(result.Errors, taskDesignError("gate-reason", path, "Gate Map", "Reason Code", fmt.Sprintf("invalid human gate reason %q", row[4])))
		} else if row[3] == "no" && row[4] != "NONE" {
			result.Errors = append(result.Errors, taskDesignError("gate-reason", path, "Gate Map", "Reason Code", "non-human gate must use NONE"))
		}
		gates = append(gates, taskDesignGate{Package: packageID, Values: append([]string(nil), row...)})
	}
	return gates
}

func lintTaskDesignBudget(path string, rows [][]string, gates []taskDesignGate, result *taskDesignLintResult) map[string]string {
	budget := map[string]string{}
	if len(rows) != len(taskDesignBudgetKeys) {
		result.Errors = append(result.Errors, taskDesignError("budget-keys", path, "Complexity Budget", "rows", "expected exactly five ordered keys"))
	}
	for index, key := range taskDesignBudgetKeys {
		if index >= len(rows) || len(rows[index]) != 2 || rows[index][0] != key {
			result.Errors = append(result.Errors, taskDesignError("budget-keys", path, "Complexity Budget", "Key", fmt.Sprintf("expected key %s at row %d", key, index+1)))
			continue
		}
		budget[key] = rows[index][1]
		if index < 4 && !taskDesignUnsignedPattern.MatchString(rows[index][1]) {
			result.Errors = append(result.Errors, taskDesignError("budget-integer", path, "Complexity Budget", key, "value must be an unsigned decimal integer"))
		}
	}
	humanCount := 0
	gateByID := map[int]taskDesignGate{}
	for _, gate := range gates {
		gateByID[gate.Package] = gate
		if gate.Values[3] == "yes" {
			humanCount++
		}
	}
	if value, ok := budget["human_gates"]; ok && taskDesignUnsignedPattern.MatchString(value) {
		count, _ := strconv.Atoi(value)
		if count != humanCount {
			result.Errors = append(result.Errors, taskDesignError("budget-human-gates", path, "Complexity Budget", "human_gates", "value must equal Human=yes Gate Map rows"))
		}
	}
	if scope, ok := budget["continuous_automation_scope"]; ok {
		lintTaskDesignSelector(path, scope, gateByID, result)
	}
	return budget
}

func lintTaskDesignSelector(path, scope string, gates map[int]taskDesignGate, result *taskDesignLintResult) {
	if scope == "packages:none" {
		return
	}
	if !strings.HasPrefix(scope, "packages:") {
		result.Errors = append(result.Errors, taskDesignError("budget-selector", path, "Complexity Budget", "continuous_automation_scope", "selector must start with packages:"))
		return
	}
	selector := strings.TrimPrefix(scope, "packages:")
	last := 0
	seen := map[int]bool{}
	for _, token := range strings.Split(selector, ",") {
		bounds := strings.Split(token, "-")
		if len(bounds) > 2 || len(bounds) == 0 {
			result.Errors = append(result.Errors, taskDesignError("budget-selector", path, "Complexity Budget", "continuous_automation_scope", "invalid selector token"))
			return
		}
		start, err := strconv.Atoi(bounds[0])
		end := start
		if len(bounds) == 2 {
			end, err = strconv.Atoi(bounds[1])
		}
		if err != nil || start < 1 || end < start || strconv.Itoa(start) != bounds[0] || (len(bounds) == 2 && strconv.Itoa(end) != bounds[1]) {
			result.Errors = append(result.Errors, taskDesignError("budget-selector", path, "Complexity Budget", "continuous_automation_scope", "selector IDs and ranges must be positive and ascending"))
			return
		}
		for id := start; id <= end; id++ {
			gate, ok := gates[id]
			if !ok || gate.Values[3] != "no" || seen[id] || id <= last {
				result.Errors = append(result.Errors, taskDesignError("budget-selector", path, "Complexity Budget", "continuous_automation_scope", fmt.Sprintf("package %d is missing, human, duplicate, or out of order", id)))
				return
			}
			seen[id] = true
			last = id
		}
	}
}

func lintTaskDesignPackageMapping(path string, gates []taskDesignGate, packages []int, result *taskDesignLintResult) {
	if len(gates) != len(packages) {
		result.Errors = append(result.Errors, taskDesignError("package-mapping", path, "Gate Map", "Package", "Gate Map and numbered package headings must have equal rows"))
		return
	}
	for index := range gates {
		if gates[index].Package != packages[index] {
			result.Errors = append(result.Errors, taskDesignError("package-mapping", path, "Gate Map", "Package", "Gate Map package order must match numbered package headings"))
			return
		}
	}
}

func lintTaskDesignLayers(path string, rows [][]string, expected int, gates []taskDesignGate, result *taskDesignLintResult) {
	if len(rows) != expected {
		result.Errors = append(result.Errors, taskDesignError("stateful-count", path, "Stateful Layer Map", "rows", fmt.Sprintf("expected %d layer rows, got %d", expected, len(rows))))
	}
	gateByID := map[int]taskDesignGate{}
	for _, gate := range gates {
		gateByID[gate.Package] = gate
	}
	layers := map[string]bool{}
	orders := map[int]int{}
	baselines := map[string]struct {
		environment string
		packageID   int
		order       int
	}{}
	for index, row := range rows {
		if len(row) != len(taskDesignLayerHeader) {
			result.Errors = append(result.Errors, taskDesignError("stateful-row", path, "Stateful Layer Map", "row", fmt.Sprintf("row %d has %d columns", index+1, len(row))))
			continue
		}
		if !taskDesignNamePattern.MatchString(row[0]) || layers[row[0]] {
			result.Errors = append(result.Errors, taskDesignError("stateful-layer", path, "Stateful Layer Map", "Layer", fmt.Sprintf("invalid or duplicate layer %q", row[0])))
		}
		layers[row[0]] = true
		packageID, packageErr := strconv.Atoi(row[1])
		gate, gateOK := gateByID[packageID]
		if packageErr != nil || !gateOK || (gate.Values[2] != "R2" && gate.Values[2] != "R3") {
			result.Errors = append(result.Errors, taskDesignError("stateful-package", path, "Stateful Layer Map", "Package", "layer must reference an R2 or R3 package"))
		}
		validEnvironment := row[2] == "local" || row[2] == "shared-local" || row[2] == "uat" || row[2] == "prod"
		if !validEnvironment {
			result.Errors = append(result.Errors, taskDesignError("stateful-environment", path, "Stateful Layer Map", "Environment", fmt.Sprintf("invalid environment %q", row[2])))
		}
		order, orderErr := strconv.Atoi(row[3])
		orders[packageID]++
		if orderErr != nil || order != orders[packageID] {
			result.Errors = append(result.Errors, taskDesignError("stateful-order", path, "Stateful Layer Map", "Order", "orders must start at 1 and be continuous per package"))
		}
		for _, column := range []int{4, 5, 8, 9, 10, 11} {
			if row[column] == "" {
				result.Errors = append(result.Errors, taskDesignError("stateful-field", path, "Stateful Layer Map", taskDesignLayerHeader[column], "value must be non-empty"))
			}
		}
		if row[6] != "backup" && row[6] != "approved-disposable-recovery" {
			result.Errors = append(result.Errors, taskDesignError("stateful-recovery", path, "Stateful Layer Map", "Recovery Evidence", "expected backup or approved-disposable-recovery"))
		}
		if row[6] == "approved-disposable-recovery" && (row[2] != "local" || (gateOK && gate.Values[2] != "R2")) {
			result.Errors = append(result.Errors, taskDesignError("stateful-recovery", path, "Stateful Layer Map", "Recovery Evidence", "disposable recovery is only valid for local R2"))
		}
		parts := strings.Split(row[7], ":")
		if len(parts) != 2 || (parts[0] != "new" && parts[0] != "reuse") || !taskDesignNamePattern.MatchString(parts[len(parts)-1]) {
			result.Errors = append(result.Errors, taskDesignError("stateful-baseline", path, "Stateful Layer Map", "Recovery Baseline", "expected new:kebab-id or reuse:kebab-id"))
		} else if parts[0] == "new" {
			if _, exists := baselines[parts[1]]; exists {
				result.Errors = append(result.Errors, taskDesignError("stateful-baseline", path, "Stateful Layer Map", "Recovery Baseline", "new baseline ID must be unique"))
			} else {
				baselines[parts[1]] = struct {
					environment string
					packageID   int
					order       int
				}{row[2], packageID, order}
			}
		} else {
			baseline, exists := baselines[parts[1]]
			before := strings.ToLower(row[9])
			for _, required := range []string{"identity", "scope", "count", "hash", "schema"} {
				if !strings.Contains(before, required) {
					exists = false
				}
			}
			if !exists || baseline.environment != row[2] || baseline.packageID != packageID || baseline.order >= order {
				result.Errors = append(result.Errors, taskDesignError("stateful-baseline", path, "Stateful Layer Map", "Recovery Baseline", "reuse must reference an earlier baseline in the same package/environment and reassert identity/scope/count/hash/schema"))
			}
		}
		expectedPattern := regexp.MustCompile(`^counts=[^;]+;hash=[^;]+;schema=[^;]+$`)
		if !expectedPattern.MatchString(row[8]) {
			result.Errors = append(result.Errors, taskDesignError("stateful-expected", path, "Stateful Layer Map", "Expected Counts/Hash/Schema", "expected counts=<value>;hash=<value>;schema=<value>"))
		}
	}
}

func compareTaskDesignArtifacts(proposal, tasks taskDesignArtifact, result *taskDesignLintResult) {
	if !equalTaskDesignGateRows(proposal.Gates, tasks.Gates) {
		result.Errors = append(result.Errors, taskDesignError("gate-mismatch", tasks.Path, "Gate Map", "rows", "Proposal and tasks Gate Map must match"))
	}
	if !equalTaskDesignRows(proposal.BudgetRows, tasks.BudgetRows) {
		result.Errors = append(result.Errors, taskDesignError("budget-mismatch", tasks.Path, "Complexity Budget", "rows", "Proposal and tasks Complexity Budget must match"))
	}
	if !equalTaskDesignRows(proposal.Layers, tasks.Layers) {
		result.Errors = append(result.Errors, taskDesignError("stateful-mismatch", tasks.Path, "Stateful Layer Map", "rows", "Proposal and tasks Stateful Layer Map must match"))
	}
}

func lintTaskDesignMicroPackages(artifact taskDesignArtifact, result *taskDesignLintResult) {
	terms := []string{"test", "测试", "dry-run", "commit", "push", "checkpoint"}
	for _, gate := range artifact.Gates {
		if gate.Values[3] != "no" {
			continue
		}
		value := strings.ToLower(gate.Values[1])
		for _, term := range terms {
			if strings.Contains(value, term) {
				result.Warnings = append(result.Warnings, taskDesignWarning("micro-package", artifact.Path, "Gate Map", strconv.Itoa(gate.Package), fmt.Sprintf("package %d may elevate %q into a top-level package; review cohesion", gate.Package, term)))
				break
			}
		}
	}
	if len(artifact.Gates) > 6 {
		result.Warnings = append(result.Warnings, taskDesignWarning("package-complexity", artifact.Path, "Gate Map", "rows", "more than six packages; review whether risk boundaries are over-split"))
	}
}

func readTaskDesignBaseline(root string) (map[string][]string, taskDesignLintResult) {
	result := taskDesignLintResult{}
	path := baselinePath(root)
	content, err := os.ReadFile(path)
	if err != nil {
		result.Errors = append(result.Errors, taskDesignError("baseline-format", path, "baseline", "file", err.Error()))
		return nil, result
	}
	lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
	if len(lines) == 0 || lines[0] != "change_name\treason" {
		result.Errors = append(result.Errors, taskDesignError("baseline-format", path, "baseline", "header", "expected change_name<TAB>reason"))
		return nil, result
	}
	rows := map[string][]string{}
	for index, line := range lines[1:] {
		columns := strings.Split(line, "\t")
		if len(columns) != 2 || !taskDesignNamePattern.MatchString(columns[0]) || strings.TrimSpace(columns[1]) == "" || strings.Contains(columns[1], "\r") {
			result.Errors = append(result.Errors, taskDesignError("baseline-format", path, "baseline", fmt.Sprintf("row %d", index+2), "expected valid kebab-case name and one non-empty reason"))
			continue
		}
		rows[columns[0]] = append(rows[columns[0]], columns[1])
	}
	return rows, result
}

func readTaskDesignArchivedNames(archiveDir string) map[string]bool {
	archived := map[string]bool{}
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		return archived
	}
	datePrefix := regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}-`)
	for _, entry := range entries {
		if entry.IsDir() {
			archived[datePrefix.ReplaceAllString(entry.Name(), "")] = true
		}
	}
	return archived
}

func splitTaskDesignSections(content string) []taskDesignSection {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	sections := []taskDesignSection{}
	current := -1
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			sections = append(sections, taskDesignSection{Name: strings.TrimSpace(strings.TrimPrefix(line, "## "))})
			current = len(sections) - 1
			continue
		}
		if current >= 0 {
			sections[current].Content += line + "\n"
		}
	}
	return sections
}

func taskDesignSectionIndex(sections []taskDesignSection, name string) int {
	for index, section := range sections {
		if section.Name == name {
			return index
		}
	}
	return -1
}

func parseTaskDesignTable(content string) (taskDesignTable, bool) {
	lines := strings.Split(content, "\n")
	tableLines := []string{}
	started := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
			started = true
			tableLines = append(tableLines, line)
		} else if started && line != "" {
			break
		}
	}
	if len(tableLines) < 2 {
		return taskDesignTable{}, false
	}
	header := splitTaskDesignTableRow(tableLines[0])
	separator := splitTaskDesignTableRow(tableLines[1])
	if len(separator) != len(header) {
		return taskDesignTable{}, false
	}
	for _, cell := range separator {
		if !regexp.MustCompile(`^:?-{3,}:?$`).MatchString(cell) {
			return taskDesignTable{}, false
		}
	}
	rows := make([][]string, 0, len(tableLines)-2)
	for _, line := range tableLines[2:] {
		rows = append(rows, splitTaskDesignTableRow(line))
	}
	return taskDesignTable{Header: header, Rows: rows}, true
}

func splitTaskDesignTableRow(line string) []string {
	line = strings.TrimSpace(strings.Trim(line, "|"))
	parts := strings.Split(line, "|")
	for index := range parts {
		parts[index] = strings.Join(strings.Fields(parts[index]), " ")
	}
	return parts
}

func equalTaskDesignStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalTaskDesignRows(left, right [][]string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !equalTaskDesignStrings(left[index], right[index]) {
			return false
		}
	}
	return true
}

func equalTaskDesignGateRows(left, right []taskDesignGate) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !equalTaskDesignStrings(left[index].Values, right[index].Values) {
			return false
		}
	}
	return true
}

func taskDesignError(code, path, section, field, message string) taskDesignDiagnostic {
	return taskDesignDiagnostic{Code: code, Path: path, Section: section, Field: field, Message: message}
}

func taskDesignWarning(code, path, section, field, message string) taskDesignDiagnostic {
	return taskDesignDiagnostic{Code: code, Path: path, Section: section, Field: field, Message: message}
}

func baselinePath(root string) string {
	return filepath.Join(root, ".agents", "openspec-task-lint-baseline.tsv")
}

func (result *taskDesignLintResult) merge(other taskDesignLintResult) {
	result.Errors = append(result.Errors, other.Errors...)
	result.Warnings = append(result.Warnings, other.Warnings...)
}
