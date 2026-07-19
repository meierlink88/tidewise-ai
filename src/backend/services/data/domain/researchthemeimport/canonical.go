package researchthemeimport

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"unicode/utf8"
)

// CanonicalHash applies the V1 RFC 8785-compatible representation to the
// validated, typed request. V1 object keys are an ASCII-only fixed contract;
// array order remains significant and is validated before this function runs.
func CanonicalHash(batch Batch) (string, error) {
	payload, err := json.Marshal(batch)
	if err != nil {
		return "", fmt.Errorf("encode research theme publication batch: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", fmt.Errorf("decode research theme publication batch: %w", err)
	}

	var canonical bytes.Buffer
	if err := writeCanonicalJSON(&canonical, value); err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

func writeCanonicalJSON(writer *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		writer.WriteString("null")
	case string:
		return writeCanonicalString(writer, typed)
	case []any:
		writer.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				writer.WriteByte(',')
			}
			if err := writeCanonicalJSON(writer, item); err != nil {
				return err
			}
		}
		writer.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		writer.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				writer.WriteByte(',')
			}
			if err := writeCanonicalString(writer, key); err != nil {
				return err
			}
			writer.WriteByte(':')
			if err := writeCanonicalJSON(writer, typed[key]); err != nil {
				return err
			}
		}
		writer.WriteByte('}')
	default:
		return fmt.Errorf("unsupported V1 canonical JSON value %T", value)
	}
	return nil
}

func writeCanonicalString(writer *bytes.Buffer, value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("canonical JSON string contains invalid UTF-8")
	}
	const hexadecimal = "0123456789abcdef"
	writer.WriteByte('"')
	for index := 0; index < len(value); index++ {
		character := value[index]
		switch character {
		case '"', '\\':
			writer.WriteByte('\\')
			writer.WriteByte(character)
		case '\b':
			writer.WriteString(`\b`)
		case '\t':
			writer.WriteString(`\t`)
		case '\n':
			writer.WriteString(`\n`)
		case '\f':
			writer.WriteString(`\f`)
		case '\r':
			writer.WriteString(`\r`)
		default:
			if character < 0x20 {
				writer.WriteString(`\u00`)
				writer.WriteByte(hexadecimal[character>>4])
				writer.WriteByte(hexadecimal[character&0x0f])
				continue
			}
			writer.WriteByte(character)
		}
	}
	writer.WriteByte('"')
	return nil
}
