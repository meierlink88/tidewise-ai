// Package materialization owns the deterministic Collector rules that decide
// how Connector Candidates are merged, identified, and classified. Files,
// indexes, and atomic publication remain Data adapter concerns.
package materialization

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/text/unicode/norm"
)

var tokenPattern = regexp.MustCompile(`[\pL\pN_]+`)
var excessNewlinePattern = regexp.MustCompile(`\n{3,}`)

type ExistingRecord struct {
	URLHash     string
	ContentHash string
	SimHash     string
}

type Decision struct {
	Candidate     collector.Candidate
	Disposition   collector.CandidateDisposition
	PublishedAt   time.Time
	PublishedText string
	TimeBasis     string
	URLHash       string
	ContentHash   string
	SimHash       string
	DocumentID    string
}

func Evaluate(
	item collector.Candidate,
	collectedAt time.Time,
	windowStart time.Time,
	records []ExistingRecord,
	nearDuplicateRadius int,
) Decision {
	item.Title = strings.TrimSpace(item.Title)
	item.URL = CanonicalURL(item.URL)
	item.Content = strings.TrimSpace(item.Content)
	if item.Content == "" {
		item.Content = item.Title
	}
	decision := Decision{Candidate: item, Disposition: collector.DispositionInvalidResult}
	if item.Title == "" || item.URL == "" || !ValidContentLevel(item.ContentLevel) {
		return decision
	}
	published := ParseTime(item.PublishedAtHint)
	if !published.IsZero() && (published.Before(windowStart) || published.After(collectedAt)) {
		decision.Disposition = collector.DispositionOutOfWindow
		return decision
	}
	decision.URLHash = Hash(item.URL)
	for _, record := range records {
		if record.URLHash == decision.URLHash {
			decision.Disposition = collector.DispositionKnownURL
			return decision
		}
	}
	body := NormalizeBody("# " + item.Title + "\n\n" + item.Content)
	decision.ContentHash = Hash(body)
	decision.SimHash = SimHash(body)
	for _, record := range records {
		if record.ContentHash == decision.ContentHash {
			decision.Disposition = collector.DispositionExactDuplicate
			return decision
		}
		if nearDuplicateRadius > 0 && Hamming(record.SimHash, decision.SimHash) <= nearDuplicateRadius {
			decision.Disposition = collector.DispositionNearDuplicate
			return decision
		}
	}
	decision.Disposition = collector.DispositionAccepted
	decision.PublishedAt = collectedAt
	decision.TimeBasis = "collected_at"
	if !published.IsZero() {
		decision.PublishedAt = published
		decision.PublishedText = published.Format(time.RFC3339)
		decision.TimeBasis = "published_at"
	}
	decision.DocumentID = "sha256:" + Hash(item.URL+"\n"+decision.ContentHash)
	return decision
}

func Reason(disposition collector.CandidateDisposition) string {
	switch disposition {
	case collector.DispositionAccepted:
		return "accepted"
	case collector.DispositionKnownURL:
		return "canonical_url_already_indexed"
	case collector.DispositionOutOfWindow:
		return "published_at_outside_time_window"
	case collector.DispositionInvalidResult:
		return "missing_title_or_url_or_content_level"
	case collector.DispositionExactDuplicate:
		return "content_sha256_already_indexed"
	case collector.DispositionNearDuplicate:
		return "simhash_within_radius"
	default:
		return "unknown_disposition"
	}
}

func Merge(items []collector.Candidate) []collector.Candidate {
	groups := make(map[string][]collector.Candidate)
	var invalid int
	for _, item := range items {
		key := CanonicalURL(item.URL)
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
			if richer(item, best) {
				best = item
			}
		}
		best.URL = key
		best.PrimaryConnector = best.Connector
		best.Connectors = best.Connectors[:0]
		for name := range connectorSet {
			best.Connectors = append(best.Connectors, name)
		}
		sort.Strings(best.Connectors)
		merged = append(merged, best)
	}
	return merged
}

func CanonicalURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	hostname := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if port == "" || parsed.Scheme == "https" && port == "443" || parsed.Scheme == "http" && port == "80" {
		if strings.Contains(hostname, ":") {
			parsed.Host = "[" + hostname + "]"
		} else {
			parsed.Host = hostname
		}
	} else {
		parsed.Host = net.JoinHostPort(hostname, port)
	}
	parsed.Fragment = ""
	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" ||
			lower == "ref" || lower == "mc_cid" || lower == "mc_eid" {
			query.Del(key)
			continue
		}
		sort.Strings(query[key])
	}
	parsed.RawQuery = query.Encode()
	if parsed.Path == "" {
		parsed.Path = "/"
	} else if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
		if parsed.Path == "" {
			parsed.Path = "/"
		}
	}
	return parsed.String()
}

func Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func SimHash(value string) string {
	weights := make([]int, 64)
	counts := make(map[string]int)
	for _, token := range tokenPattern.FindAllString(strings.ToLower(norm.NFKC.String(value)), -1) {
		counts[token]++
	}
	if len(counts) == 0 {
		return "0000000000000000"
	}
	for token, count := range counts {
		digest, _ := blake2b.New(8, nil)
		_, _ = digest.Write([]byte(token))
		bits := binary.BigEndian.Uint64(digest.Sum(nil))
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

func NormalizeBody(value string) string {
	value = norm.NFC.String(strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n"))
	lines := strings.Split(value, "\n")
	for index := range lines {
		lines[index] = strings.TrimRightFunc(lines[index], unicode.IsSpace)
	}
	return excessNewlinePattern.ReplaceAllString(strings.TrimSpace(strings.Join(lines, "\n")), "\n\n")
}

func Hamming(left, right string) int {
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

func ParseTime(value string) time.Time {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC1123,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		parsed, err := time.Parse(layout, strings.TrimSpace(value))
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func ValidContentLevel(level collector.ContentLevel) bool {
	return rank(level) >= 0
}

func richer(candidate, current collector.Candidate) bool {
	if candidateRank, currentRank := rank(candidate.ContentLevel), rank(current.ContentLevel); candidateRank != currentRank {
		return candidateRank > currentRank
	}
	if candidateLength, currentLength := nonWhitespaceRunes(candidate.Content), nonWhitespaceRunes(current.Content); candidateLength != currentLength {
		return candidateLength > currentLength
	}
	return stableCandidateKey(candidate) < stableCandidateKey(current)
}

func nonWhitespaceRunes(value string) int {
	count := 0
	for _, character := range value {
		if !unicode.IsSpace(character) {
			count++
		}
	}
	return count
}

func stableCandidateKey(item collector.Candidate) string {
	connectorOrder := len(collector.ConnectorKeys())
	for index, connector := range collector.ConnectorKeys() {
		if item.Connector == connector {
			connectorOrder = index
			break
		}
	}
	return fmt.Sprintf("%03d\x00%09d\x00%s\x00%s\x00%s",
		connectorOrder, item.ResultPosition, item.SourceExternalID, item.Content, item.Title)
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
