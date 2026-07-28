package researchreasoningtreeimport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type DecodeError struct {
	Path    string
	Message string
}

func (e *DecodeError) Error() string {
	if e == nil {
		return "Research Reason Tree request does not match the V1 JSON contract"
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

func decodeStrictJSON(reader io.Reader) (Publication, error) {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return Publication{}, err
	}
	if !utf8.Valid(payload) {
		return Publication{}, &DecodeError{Path: "$", Message: "must contain valid UTF-8"}
	}
	shapeDecoder := json.NewDecoder(bytes.NewReader(payload))
	if err := validateJSONValue(shapeDecoder, "$"); err != nil {
		return Publication{}, err
	}
	if token, err := shapeDecoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return Publication{}, fmt.Errorf("request body must contain one JSON object; trailing token %v", token)
		}
		return Publication{}, fmt.Errorf("decode trailing Research Reason Tree publication data: %w", err)
	}

	typed := json.NewDecoder(bytes.NewReader(payload))
	typed.DisallowUnknownFields()
	var publication Publication
	if err := typed.Decode(&publication); err != nil {
		return Publication{}, fmt.Errorf("decode Research Reason Tree publication: %w", err)
	}
	return publication, nil
}

func validateJSONValue(decoder *json.Decoder, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode Research Reason Tree publication: %w", err)
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key is %T, want string", keyToken)
			}
			childPath := path + "." + key
			if _, duplicate := seen[key]; duplicate {
				return &DecodeError{Path: childPath, Message: "must not be duplicated"}
			}
			seen[key] = struct{}{}
			if err := validateJSONValue(decoder, childPath); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	case '[':
		index := 0
		for decoder.More() {
			if err := validateJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
			index++
		}
		_, err := decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}
