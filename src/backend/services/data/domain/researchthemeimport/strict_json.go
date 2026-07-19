package researchthemeimport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type DecodeError struct {
	ThemeKey string
	Path     string
	Message  string
}

func (e *DecodeError) Error() string {
	if e == nil {
		return "research Theme request does not match the V1 JSON contract"
	}
	if e.ThemeKey != "" {
		return fmt.Sprintf("%s: %s: %s", e.ThemeKey, e.Path, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

type jsonNodeKind uint8

const (
	jsonScalar jsonNodeKind = iota
	jsonObject
	jsonArray
)

type jsonMember struct {
	name  string
	value jsonNode
}

type jsonNode struct {
	kind   jsonNodeKind
	scalar any
	object []jsonMember
	array  []jsonNode
}

type jsonShape struct {
	kind             jsonNodeKind
	fields           map[string]*jsonShape
	item             *jsonShape
	capturesThemeKey bool
}

func decodeStrictJSON(reader io.Reader) (Batch, error) {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return Batch{}, err
	}
	if !utf8.Valid(payload) {
		return Batch{}, &DecodeError{Path: "$", Message: "must contain valid UTF-8"}
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	node, err := parseJSONNode(decoder)
	if err != nil {
		return Batch{}, fmt.Errorf("decode research Theme publication batch: %w", err)
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return Batch{}, fmt.Errorf("request body must contain one JSON object; trailing token %v", token)
		}
		return Batch{}, fmt.Errorf("decode trailing research Theme publication data: %w", err)
	}
	if err := validateJSONShape(node, researchThemeBatchShape(), "", ""); err != nil {
		return Batch{}, err
	}

	typed := json.NewDecoder(bytes.NewReader(payload))
	typed.DisallowUnknownFields()
	var batch Batch
	if err := typed.Decode(&batch); err != nil {
		return Batch{}, fmt.Errorf("decode research Theme publication batch: %w", err)
	}
	return batch, nil
}

func parseJSONNode(decoder *json.Decoder) (jsonNode, error) {
	token, err := decoder.Token()
	if err != nil {
		return jsonNode{}, err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return jsonNode{kind: jsonScalar, scalar: token}, nil
	}
	switch delimiter {
	case '{':
		node := jsonNode{kind: jsonObject}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return jsonNode{}, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return jsonNode{}, fmt.Errorf("object key is %T, want string", keyToken)
			}
			value, err := parseJSONNode(decoder)
			if err != nil {
				return jsonNode{}, err
			}
			node.object = append(node.object, jsonMember{name: key, value: value})
		}
		if _, err := decoder.Token(); err != nil {
			return jsonNode{}, err
		}
		return node, nil
	case '[':
		node := jsonNode{kind: jsonArray}
		for decoder.More() {
			value, err := parseJSONNode(decoder)
			if err != nil {
				return jsonNode{}, err
			}
			node.array = append(node.array, value)
		}
		if _, err := decoder.Token(); err != nil {
			return jsonNode{}, err
		}
		return node, nil
	default:
		return jsonNode{}, fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

func validateJSONShape(node jsonNode, shape *jsonShape, path, themeKey string) error {
	if node.kind != shape.kind {
		location := path
		if location == "" {
			location = "$"
		}
		return &DecodeError{ThemeKey: themeKey, Path: location, Message: "has an invalid JSON type"}
	}
	switch shape.kind {
	case jsonObject:
		if shape.capturesThemeKey {
			themeKey = nodeStringMember(node, "theme_key")
		}
		seen := make(map[string]struct{}, len(node.object))
		for _, member := range node.object {
			memberPath := joinJSONPath(path, member.name)
			if _, duplicate := seen[member.name]; duplicate {
				return &DecodeError{ThemeKey: themeKey, Path: memberPath, Message: "must not be duplicated"}
			}
			seen[member.name] = struct{}{}
			fieldShape, exists := shape.fields[member.name]
			if !exists {
				return &DecodeError{ThemeKey: themeKey, Path: memberPath, Message: "is not part of the V1 contract"}
			}
			if err := validateJSONShape(member.value, fieldShape, memberPath, themeKey); err != nil {
				return err
			}
		}
	case jsonArray:
		for index, item := range node.array {
			if err := validateJSONShape(item, shape.item, fmt.Sprintf("%s[%d]", path, index), themeKey); err != nil {
				return err
			}
		}
	}
	return nil
}

func nodeStringMember(node jsonNode, name string) string {
	for _, member := range node.object {
		if member.name == name {
			value, _ := member.value.scalar.(string)
			return value
		}
	}
	return ""
}

func joinJSONPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

func researchThemeBatchShape() *jsonShape {
	scalar := &jsonShape{kind: jsonScalar}
	chainNode := &jsonShape{kind: jsonObject, fields: map[string]*jsonShape{
		"chain_node_id": scalar, "relation_role": scalar, "impact_summary": scalar,
	}}
	event := &jsonShape{kind: jsonObject, fields: map[string]*jsonShape{
		"event_id": scalar, "evidence_role": scalar, "supported_claim": scalar,
	}}
	theme := &jsonShape{kind: jsonObject, capturesThemeKey: true, fields: map[string]*jsonShape{
		"theme_key": scalar, "name": scalar, "one_line_conclusion": scalar,
		"impact_level": scalar, "transmission_path": scalar, "trading_direction": scalar,
		"transmission_stage": scalar, "next_checkpoint": scalar, "market_confirmation_summary": scalar,
		"chain_nodes": {kind: jsonArray, item: chainNode},
		"events":      {kind: jsonArray, item: event},
	}}
	return &jsonShape{kind: jsonObject, fields: map[string]*jsonShape{
		"analysis_batch_id": scalar,
		"window_start":      scalar,
		"window_end":        scalar,
		"themes":            {kind: jsonArray, item: theme},
	}}
}
