package report

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"unicode/utf16"
	"unicode/utf8"
)

// canonicalPayloadHash implements the RFC 8785 JSON Canonicalization Scheme
// for the value classes admitted by the current versioned Report publication API.
func canonicalPayloadHash(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return "", err
	}
	var canonical bytes.Buffer
	if err := writeCanonicalJSON(&canonical, decoded); err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical.Bytes())
	return hex.EncodeToString(digest[:]), nil
}

func writeCanonicalJSON(writer *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		writer.WriteString("null")
	case bool:
		if typed {
			writer.WriteString("true")
		} else {
			writer.WriteString("false")
		}
	case string:
		if err := writeCanonicalString(writer, typed); err != nil {
			return err
		}
	case json.Number:
		value, err := typed.Float64()
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return errors.New("Report canonical JSON contains an invalid number")
		}
		if value == 0 {
			writer.WriteByte('0')
			return nil
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		writer.Write(encoded)
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
		sort.Slice(keys, func(i, j int) bool { return utf16Less(keys[i], keys[j]) })
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
		return errors.New("Report canonical JSON contains an unsupported value")
	}
	return nil
}

func utf16Less(left, right string) bool {
	a := utf16.Encode([]rune(left))
	b := utf16.Encode([]rune(right))
	for index := 0; index < len(a) && index < len(b); index++ {
		if a[index] != b[index] {
			return a[index] < b[index]
		}
	}
	return len(a) < len(b)
}

func writeCanonicalString(writer *bytes.Buffer, value string) error {
	if !utf8.ValidString(value) {
		return errors.New("Report canonical JSON contains invalid UTF-8")
	}
	const hexDigits = "0123456789abcdef"
	writer.WriteByte('"')
	for _, current := range value {
		switch current {
		case '"', '\\':
			writer.WriteByte('\\')
			writer.WriteRune(current)
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
			if current >= 0 && current <= 0x1f {
				writer.WriteString(`\u00`)
				writer.WriteByte(hexDigits[byte(current)>>4])
				writer.WriteByte(hexDigits[byte(current)&0x0f])
			} else {
				writer.WriteRune(current)
			}
		}
	}
	writer.WriteByte('"')
	return nil
}
