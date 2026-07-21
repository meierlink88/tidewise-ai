package researchanchorimport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type DecodeError struct {
	CenterChainNodeID string
	Path              string
	Message           string
}

func (e *DecodeError) Error() string {
	if e == nil {
		return "research Anchor request does not match the V1 JSON contract"
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
	required         []string
	item             *jsonShape
	capturesCenterID bool
	allowsNull       bool
}

func decodeStrictJSON(reader io.Reader) (Publication, error) {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return Publication{}, err
	}
	if !utf8.Valid(payload) {
		return Publication{}, &DecodeError{Path: "$", Message: "must contain valid UTF-8"}
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	node, err := parseJSONNode(decoder)
	if err != nil {
		return Publication{}, fmt.Errorf("decode Research Anchor publication: %w", err)
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return Publication{}, fmt.Errorf("request body must contain one JSON object; trailing token %v", token)
		}
		return Publication{}, fmt.Errorf("decode trailing Research Anchor publication data: %w", err)
	}
	if err := validateJSONShape(node, publicationShape(), "", ""); err != nil {
		return Publication{}, err
	}
	typed := json.NewDecoder(bytes.NewReader(payload))
	typed.DisallowUnknownFields()
	var publication Publication
	if err := typed.Decode(&publication); err != nil {
		return Publication{}, fmt.Errorf("decode Research Anchor publication: %w", err)
	}
	if err := validateUUIDJSONFormat(publication); err != nil {
		return Publication{}, err
	}
	return publication, nil
}

func validateUUIDJSONFormat(publication Publication) error {
	if !uuidPattern.MatchString(publication.ThemeID) {
		return &DecodeError{Path: "theme_id", Message: "must be a standard lowercase UUID"}
	}
	for anchorIndex, anchor := range publication.Anchors {
		anchorPath := fmt.Sprintf("anchors[%d]", anchorIndex)
		if !uuidPattern.MatchString(anchor.CenterChainNodeID) {
			return &DecodeError{
				CenterChainNodeID: anchor.CenterChainNodeID,
				Path:              anchorPath + ".center_chain_node_id",
				Message:           "must be a standard lowercase UUID",
			}
		}
		for eventIndex, event := range anchor.Events {
			if !uuidPattern.MatchString(event.EventID) {
				return &DecodeError{
					CenterChainNodeID: anchor.CenterChainNodeID,
					Path:              fmt.Sprintf("%s.events[%d].event_id", anchorPath, eventIndex),
					Message:           "must be a standard lowercase UUID",
				}
			}
		}
		for nodeIndex, node := range anchor.PathNodes {
			if !uuidPattern.MatchString(node.ChainNodeID) {
				return &DecodeError{
					CenterChainNodeID: anchor.CenterChainNodeID,
					Path:              fmt.Sprintf("%s.path_nodes[%d].chain_node_id", anchorPath, nodeIndex),
					Message:           "must be a standard lowercase UUID",
				}
			}
		}
	}
	return nil
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

func validateJSONShape(node jsonNode, shape *jsonShape, path, centerID string) error {
	if node.kind != shape.kind {
		if path == "" {
			path = "$"
		}
		return &DecodeError{CenterChainNodeID: centerID, Path: path, Message: "has an invalid JSON type"}
	}
	switch shape.kind {
	case jsonScalar:
		if node.scalar == nil && shape.allowsNull {
			return nil
		}
		if _, ok := node.scalar.(string); !ok {
			return &DecodeError{CenterChainNodeID: centerID, Path: path, Message: "must be a JSON string"}
		}
	case jsonObject:
		if shape.capturesCenterID {
			centerID = nodeStringMember(node, "center_chain_node_id")
		}
		seen := make(map[string]struct{}, len(node.object))
		for _, member := range node.object {
			memberPath := joinJSONPath(path, member.name)
			if _, duplicate := seen[member.name]; duplicate {
				return &DecodeError{CenterChainNodeID: centerID, Path: memberPath, Message: "must not be duplicated"}
			}
			seen[member.name] = struct{}{}
			fieldShape, exists := shape.fields[member.name]
			if !exists {
				return &DecodeError{CenterChainNodeID: centerID, Path: memberPath, Message: "is not part of the V1 contract"}
			}
			if err := validateJSONShape(member.value, fieldShape, memberPath, centerID); err != nil {
				return err
			}
		}
		for _, field := range shape.required {
			if _, exists := seen[field]; !exists {
				return &DecodeError{
					CenterChainNodeID: centerID,
					Path:              joinJSONPath(path, field),
					Message:           "is required",
				}
			}
		}
	case jsonArray:
		for index, item := range node.array {
			if err := validateJSONShape(item, shape.item, fmt.Sprintf("%s[%d]", path, index), centerID); err != nil {
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

func publicationShape() *jsonShape {
	scalar := &jsonShape{kind: jsonScalar}
	nullableScalar := &jsonShape{kind: jsonScalar, allowsNull: true}
	event := &jsonShape{kind: jsonObject, required: []string{
		"event_id", "evidence_role", "evidence_summary",
	}, fields: map[string]*jsonShape{
		"event_id": scalar, "evidence_role": scalar, "evidence_summary": scalar,
	}}
	pathNode := &jsonShape{kind: jsonObject, required: []string{
		"chain_node_id", "change_direction", "change_summary", "impact_summary", "incoming_transmission_mechanism",
	}, fields: map[string]*jsonShape{
		"chain_node_id": scalar, "change_direction": scalar, "change_summary": scalar,
		"impact_summary": scalar, "incoming_transmission_mechanism": nullableScalar,
	}}
	anchor := &jsonShape{kind: jsonObject, capturesCenterID: true, required: []string{
		"center_chain_node_id", "one_line_conclusion", "fact_summary", "net_direction_summary",
		"support_summary", "counter_summary", "trading_direction", "next_checkpoint", "events", "path_nodes",
	}, fields: map[string]*jsonShape{
		"center_chain_node_id": scalar, "one_line_conclusion": scalar, "fact_summary": scalar,
		"net_direction_summary": scalar, "support_summary": scalar, "counter_summary": nullableScalar,
		"trading_direction": scalar, "next_checkpoint": scalar,
		"events": {kind: jsonArray, item: event}, "path_nodes": {kind: jsonArray, item: pathNode},
	}}
	return &jsonShape{kind: jsonObject, required: []string{"theme_id", "anchors"}, fields: map[string]*jsonShape{
		"theme_id": scalar, "anchors": {kind: jsonArray, item: anchor},
	}}
}
