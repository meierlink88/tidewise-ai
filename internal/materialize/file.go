package materialize

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
)

const indexHeader = "document_id\tpublished_at\turl_sha256\tcontent_sha256\tsimhash64\tdocument_path\n"

var tokenPattern = regexp.MustCompile(`[\pL\pN_]+`)

type File struct {
	Root                string
	NearDuplicateRadius int
}

type indexRecord struct {
	DocumentID, PublishedAt, URLHash, ContentHash, SimHash, Path string
}

func (f File) Materialize(_ context.Context, request collector.Request, runs map[string]collector.ConnectorRun) (*collector.Result, error) {
	root := f.Root
	if root == "" {
		root = "data"
	}
	documentsRoot := filepath.Join(root, "documents")
	indexPath := filepath.Join(root, "indexes", "dedup-index.tsv")
	runsRoot := filepath.Join(root, "runs")
	if err := os.MkdirAll(documentsRoot, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(runsRoot, 0o755); err != nil {
		return nil, err
	}

	stats := collector.Stats{
		ConnectorCounts: make(map[string]int), ConnectorErrors: make(map[string]string),
		ContentLevels: map[collector.ContentLevel]int{
			collector.LevelFullText: 0, collector.LevelSummary: 0,
			collector.LevelSnippet: 0, collector.LevelTitle: 0,
		},
	}
	var candidates []collector.Candidate
	for name, run := range runs {
		stats.ConnectorCounts[name] = len(run.Results)
		if run.Error != "" {
			stats.ConnectorErrors[name] = run.Error
		}
		candidates = append(candidates, run.Results...)
	}
	stats.RawResults = len(candidates)
	merged := merge(candidates)
	stats.MergedResults = len(merged)
	stats.ResultsPending = len(merged)

	records, err := loadIndex(indexPath)
	if err != nil {
		return nil, err
	}
	windowStart := request.CollectedAt.Add(-time.Duration(request.TimeWindowHours) * time.Hour)
	for _, item := range merged {
		stats.ContentLevels[item.ContentLevel]++
		state, record, path, err := f.materializeOne(item, request.CollectedAt, windowStart, documentsRoot, records)
		if err != nil {
			return nil, err
		}
		switch state {
		case "accepted":
			stats.Accepted++
			record.Path = path
			records = append(records, record)
		case "known_url":
			stats.KnownURL++
		case "out_of_window":
			stats.OutOfWindow++
		case "invalid_result":
			stats.InvalidResult++
		case "exact_duplicate":
			stats.ExactDuplicate++
		case "near_duplicate":
			stats.NearDuplicate++
		}
		stats.ResultsTerminal++
		stats.ResultsPending--
	}
	if err := writeIndex(indexPath, records); err != nil {
		return nil, err
	}

	stopReason := "connectors_completed"
	if len(stats.ConnectorErrors) > 0 {
		stopReason = "completed_with_connector_failures"
	}
	result := &collector.Result{
		RunID: request.RunID, StopReason: stopReason,
		Documents: documentsRoot, Index: indexPath,
		Summary: filepath.Join(runsRoot, request.RunID+"-summary.md"), Stats: stats,
	}
	if err := writeSummary(result, request); err != nil {
		return nil, err
	}
	return result, nil
}

func (f File) materializeOne(item collector.Candidate, collectedAt, windowStart time.Time, documentsRoot string, records []indexRecord) (string, indexRecord, string, error) {
	item.Title = strings.TrimSpace(item.Title)
	item.URL = canonicalURL(item.URL)
	item.Content = strings.TrimSpace(item.Content)
	if item.Content == "" {
		item.Content = item.Title
	}
	if item.Title == "" || item.URL == "" || !validLevel(item.ContentLevel) {
		return "invalid_result", indexRecord{}, "", nil
	}
	published := parseTime(item.PublishedAtHint)
	if !published.IsZero() && (published.Before(windowStart) || published.After(collectedAt)) {
		return "out_of_window", indexRecord{}, "", nil
	}
	urlHash := hash(item.URL)
	for _, record := range records {
		if record.URLHash == urlHash {
			return "known_url", indexRecord{}, "", nil
		}
	}
	body := "# " + item.Title + "\n\n" + item.Content
	contentHash := hash(body)
	fingerprint := simhash(body)
	for _, record := range records {
		if record.ContentHash == contentHash {
			return "exact_duplicate", indexRecord{}, "", nil
		}
		if f.NearDuplicateRadius > 0 && hamming(record.SimHash, fingerprint) <= f.NearDuplicateRadius {
			return "near_duplicate", indexRecord{}, "", nil
		}
	}
	stamp := collectedAt
	publishedText := ""
	timeBasis := "collected_at"
	if !published.IsZero() {
		stamp = published
		publishedText = published.Format(time.RFC3339)
		timeBasis = "published_at"
	}
	documentID := "sha256:" + hash(item.URL+"\n"+contentHash)
	path := filepath.Join(documentsRoot, stamp.Format("2006/01/02"), fmt.Sprintf("%s--%s--%s.md", stamp.UTC().Format("20060102T150405Z"), slug(item.Title), urlHash[:8]))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", indexRecord{}, "", err
	}
	markdown := render(item, documentID, contentHash, publishedText, collectedAt.UTC().Format(time.RFC3339), timeBasis)
	if err := atomicWrite(path, []byte(markdown), 0o644); err != nil {
		return "", indexRecord{}, "", err
	}
	return "accepted", indexRecord{documentID, publishedText, urlHash, contentHash, fingerprint, path}, path, nil
}

func merge(items []collector.Candidate) []collector.Candidate {
	groups := make(map[string][]collector.Candidate)
	var invalid int
	for _, item := range items {
		key := canonicalURL(item.URL)
		if key == "" {
			key = fmt.Sprintf("__invalid_%d", invalid)
			invalid++
		}
		groups[key] = append(groups[key], item)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	merged := make([]collector.Candidate, 0, len(keys))
	for _, key := range keys {
		group := groups[key]
		best := group[0]
		connectorSet := make(map[string]struct{})
		for _, item := range group {
			connectorSet[item.Connector] = struct{}{}
			if rank(item.ContentLevel) > rank(best.ContentLevel) || rank(item.ContentLevel) == rank(best.ContentLevel) && len(item.Content) > len(best.Content) {
				best = item
			}
		}
		best.URL = key
		best.PrimaryConnector = best.Connector
		for name := range connectorSet {
			best.Connectors = append(best.Connectors, name)
		}
		sort.Strings(best.Connectors)
		merged = append(merged, best)
	}
	return merged
}

func render(item collector.Candidate, documentID, contentHash, published, collected, timeBasis string) string {
	publishedYAML := "null"
	if published != "" {
		publishedYAML = strconv.Quote(published)
	}
	var connectorLines strings.Builder
	for _, name := range item.Connectors {
		fmt.Fprintf(&connectorLines, "  - %s\n", strconv.Quote(name))
	}
	return fmt.Sprintf("---\nschema_version: \"connector_result_md.v1\"\ndocument_id: %s\ntitle: %s\nsource_name: %s\nsource_type: %s\nsource_url: %s\nsource_external_id: %s\nconnectors:\n%sprimary_connector: %s\ncontent_origin: \"connector_response\"\ncontent_level: %s\npublished_at: %s\ncollected_at: %s\ntime_basis: %s\nlanguage: %s\ncontent_sha256: %s\nquality_status: \"accepted\"\nsupersedes_document_id: null\n---\n\n# %s\n\n%s\n", strconv.Quote(documentID), strconv.Quote(item.Title), yamlNullable(item.SourceName), yamlNullable(item.SourceType), strconv.Quote(item.URL), yamlNullable(item.SourceExternalID), connectorLines.String(), strconv.Quote(item.PrimaryConnector), strconv.Quote(string(item.ContentLevel)), publishedYAML, strconv.Quote(collected), strconv.Quote(timeBasis), strconv.Quote(language(item.Title+item.Content)), strconv.Quote(contentHash), item.Title, item.Content)
}

func writeSummary(result *collector.Result, request collector.Request) error {
	s := result.Stats
	content := fmt.Sprintf("# Collector Run %s\n\n- time_window_hours: %d\n- stop_reason: %s\n- raw_results: %d\n- merged_results: %d\n- results_terminal: %d\n- results_pending: %d\n- accepted: %d\n- known_url: %d\n- out_of_window: %d\n- invalid_result: %d\n- exact_duplicate: %d\n- near_duplicate: %d\n- documents: %s\n- index: %s\n", request.RunID, request.TimeWindowHours, result.StopReason, s.RawResults, s.MergedResults, s.ResultsTerminal, s.ResultsPending, s.Accepted, s.KnownURL, s.OutOfWindow, s.InvalidResult, s.ExactDuplicate, s.NearDuplicate, result.Documents, result.Index)
	return atomicWrite(result.Summary, []byte(content), 0o644)
}

func loadIndex(path string) ([]indexRecord, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var records []indexRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "document_id\t") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 6 {
			return nil, fmt.Errorf("invalid index row")
		}
		records = append(records, indexRecord{parts[0], parts[1], parts[2], parts[3], parts[4], parts[5]})
	}
	return records, scanner.Err()
}

func writeIndex(path string, records []indexRecord) error {
	var builder strings.Builder
	builder.WriteString(indexHeader)
	for _, record := range records {
		fmt.Fprintf(&builder, "%s\t%s\t%s\t%s\t%s\t%s\n", record.DocumentID, record.PublishedAt, record.URLHash, record.ContentHash, record.SimHash, record.Path)
	}
	return atomicWrite(path, []byte(builder.String()), 0o644)
}

func atomicWrite(path string, content []byte, mode os.FileMode) error {
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, content, mode); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func canonicalURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" || lower == "ref" {
			query.Del(key)
		}
	}
	parsed.RawQuery = query.Encode()
	if parsed.Path != "/" {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	}
	return parsed.String()
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func simhash(value string) string {
	weights := make([]int, 64)
	counts := make(map[string]int)
	for _, token := range tokenPattern.FindAllString(strings.ToLower(value), -1) {
		counts[token]++
	}
	for token, count := range counts {
		sum := sha256.Sum256([]byte(token))
		bits := binary.BigEndian.Uint64(sum[:8])
		for bit := range 64 {
			if bits&(uint64(1)<<bit) != 0 {
				weights[bit] += count
			} else {
				weights[bit] -= count
			}
		}
	}
	var result uint64
	for bit, weight := range weights {
		if weight >= 0 {
			result |= uint64(1) << bit
		}
	}
	return fmt.Sprintf("%016x", result)
}

func hamming(left, right string) int {
	a, errA := strconv.ParseUint(left, 16, 64)
	b, errB := strconv.ParseUint(right, 16, 64)
	if errA != nil || errB != nil {
		return 65
	}
	value := a ^ b
	count := 0
	for value != 0 {
		value &= value - 1
		count++
	}
	return count
}

func parseTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, strings.TrimSpace(value))
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func rank(level collector.ContentLevel) int {
	switch level {
	case collector.LevelFullText:
		return 3
	case collector.LevelSummary:
		return 2
	case collector.LevelSnippet:
		return 1
	case collector.LevelTitle:
		return 0
	default:
		return -1
	}
}

func validLevel(level collector.ContentLevel) bool { return rank(level) >= 0 }

func slug(value string) string {
	var output []rune
	dash := false
	for _, char := range strings.ToLower(value) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			output = append(output, char)
			dash = false
		} else if !dash && len(output) > 0 {
			output = append(output, '-')
			dash = true
		}
		if len(output) >= 64 {
			break
		}
	}
	result := strings.Trim(string(output), "-")
	if result == "" {
		return "connector-result"
	}
	return result
}

func yamlNullable(value string) string {
	if strings.TrimSpace(value) == "" {
		return "null"
	}
	return strconv.Quote(value)
}

func language(value string) string {
	for _, char := range value {
		if unicode.In(char, unicode.Han) {
			return "zh"
		}
	}
	return "en"
}
