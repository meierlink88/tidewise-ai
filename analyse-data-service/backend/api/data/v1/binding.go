package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type bindingShape struct {
	fields   map[string]*bindingShape
	item     *bindingShape
	any      bool
	string   bool
	null     bool
	required []string
}

var scalarShape = &bindingShape{}
var anyShape = &bindingShape{any: true}
var stringShape = &bindingShape{string: true}
var nullableStringShape = &bindingShape{string: true, null: true}

var lowerUUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func objectShape(fields map[string]*bindingShape) *bindingShape {
	return &bindingShape{fields: fields}
}

func requiredObjectShape(required []string, fields map[string]*bindingShape) *bindingShape {
	return &bindingShape{fields: fields, required: required}
}

func arrayShape(item *bindingShape) *bindingShape {
	return &bindingShape{item: item}
}

func decodeEventPublication(payload []byte) (*EventPublicationRequest, error) {
	request := new(EventPublicationRequest)
	if err := decodeStrictBinding(payload, eventPublicationShape(), request); err != nil {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	return request, nil
}

func decodeResearchThemeImport(payload []byte) (*ResearchThemeImportRequest, error) {
	request := new(ResearchThemeImportRequest)
	if err := decodeStrictBinding(payload, researchThemeImportShape(), request); err != nil {
		path := bindingErrorPath(err)
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for the Research Theme V1 contract", map[string]any{
			"theme_key": researchThemeKeyAtPath(payload, path),
			"path":      path,
		})
	}
	return request, nil
}

func decodeResearchAnchorImport(payload []byte) (*ResearchAnchorImportRequest, error) {
	request := new(ResearchAnchorImportRequest)
	if err := decodeStrictBinding(payload, researchAnchorImportShape(), request); err != nil {
		path := bindingErrorPath(err)
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for the Research Anchor V1 contract", map[string]any{
			"center_chain_node_id": researchAnchorCenterAtPath(payload, path),
			"path":                 path,
			"reference":            "",
		})
	}
	if path, centerID := validateResearchAnchorUUIDs(request); path != "" {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for the Research Anchor V1 contract", map[string]any{
			"center_chain_node_id": centerID,
			"path":                 path,
			"reference":            "",
		})
	}
	return request, nil
}

type bindingError struct {
	path string
	err  error
}

func (e *bindingError) Error() string {
	return fmt.Sprintf("%s: %v", e.path, e.err)
}

func bindingErrorPath(err error) string {
	if failure, ok := err.(*bindingError); ok {
		return failure.path
	}
	return ""
}

func decodeStrictBinding(payload []byte, shape *bindingShape, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := validateBindingValue(decoder, shape, ""); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return &bindingError{err: fmt.Errorf("body must contain exactly one JSON value")}
		}
		return &bindingError{err: err}
	}
	decoder = json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &bindingError{path: jsonDecodeErrorPath(err), err: err}
	}
	return nil
}

func validateBindingValue(decoder *json.Decoder, shape *bindingShape, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return &bindingError{path: path, err: err}
	}
	delim, composite := token.(json.Delim)
	if shape.any {
		return consumeBindingComposite(decoder, delim, composite, path)
	}
	if shape.fields != nil {
		if !composite || delim != '{' {
			return &bindingError{path: path, err: fmt.Errorf("must be an object")}
		}
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return &bindingError{path: path, err: err}
			}
			key, ok := keyToken.(string)
			if !ok {
				return &bindingError{path: path, err: fmt.Errorf("object key must be a string")}
			}
			childPath := joinBindingPath(path, key)
			if _, duplicate := seen[key]; duplicate {
				return &bindingError{path: childPath, err: fmt.Errorf("duplicate field")}
			}
			seen[key] = struct{}{}
			child, known := shape.fields[key]
			if !known {
				return &bindingError{path: childPath, err: fmt.Errorf("unknown field")}
			}
			if err := validateBindingValue(decoder, child, childPath); err != nil {
				return err
			}
		}
		for _, field := range shape.required {
			if _, exists := seen[field]; !exists {
				return &bindingError{path: joinBindingPath(path, field), err: fmt.Errorf("required field is missing")}
			}
		}
		if _, err := decoder.Token(); err != nil {
			return &bindingError{path: path, err: err}
		}
		return nil
	}
	if shape.item != nil {
		if !composite || delim != '[' {
			return &bindingError{path: path, err: fmt.Errorf("must be an array")}
		}
		index := 0
		for decoder.More() {
			childPath := path + "[" + strconv.Itoa(index) + "]"
			if err := validateBindingValue(decoder, shape.item, childPath); err != nil {
				return err
			}
			index++
		}
		if _, err := decoder.Token(); err != nil {
			return &bindingError{path: path, err: err}
		}
		return nil
	}
	if shape.string {
		if token == nil && shape.null {
			return nil
		}
		if _, ok := token.(string); !ok {
			return &bindingError{path: path, err: fmt.Errorf("must be a JSON string")}
		}
		return nil
	}
	if composite {
		if delim == '{' || delim == '[' {
			return &bindingError{path: path, err: fmt.Errorf("must be a scalar")}
		}
	}
	return nil
}

func consumeBindingComposite(decoder *json.Decoder, delim json.Delim, composite bool, path string) error {
	if !composite {
		return nil
	}
	switch delim {
	case '{':
		for decoder.More() {
			if _, err := decoder.Token(); err != nil {
				return &bindingError{path: path, err: err}
			}
			if err := validateBindingValue(decoder, anyShape, path); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := validateBindingValue(decoder, anyShape, path); err != nil {
				return err
			}
		}
	default:
		return &bindingError{path: path, err: fmt.Errorf("unexpected delimiter")}
	}
	if _, err := decoder.Token(); err != nil {
		return &bindingError{path: path, err: err}
	}
	return nil
}

func joinBindingPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

func jsonDecodeErrorPath(err error) string {
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		return typeError.Field
	}
	return ""
}

func researchThemeKeyAtPath(payload []byte, path string) string {
	var hint struct {
		Themes []struct {
			ThemeKey string `json:"theme_key"`
		} `json:"themes"`
	}
	_ = json.Unmarshal(payload, &hint)
	index, ok := indexedBindingPath(path, "themes")
	if ok && index < len(hint.Themes) {
		return hint.Themes[index].ThemeKey
	}
	return ""
}

func researchAnchorCenterAtPath(payload []byte, path string) string {
	var hint struct {
		Anchors []struct {
			CenterChainNodeID string `json:"center_chain_node_id"`
		} `json:"anchors"`
	}
	_ = json.Unmarshal(payload, &hint)
	index, ok := indexedBindingPath(path, "anchors")
	if ok && index < len(hint.Anchors) {
		return hint.Anchors[index].CenterChainNodeID
	}
	return ""
}

func validateResearchAnchorUUIDs(request *ResearchAnchorImportRequest) (string, string) {
	if !lowerUUIDPattern.MatchString(request.ThemeID) {
		return "theme_id", ""
	}
	for anchorIndex, anchor := range request.Anchors {
		anchorPath := "anchors[" + strconv.Itoa(anchorIndex) + "]"
		if !lowerUUIDPattern.MatchString(anchor.CenterChainNodeID) {
			return anchorPath + ".center_chain_node_id", anchor.CenterChainNodeID
		}
		for eventIndex, event := range anchor.Events {
			if !lowerUUIDPattern.MatchString(event.EventID) {
				return fmt.Sprintf("%s.events[%d].event_id", anchorPath, eventIndex), anchor.CenterChainNodeID
			}
		}
		for nodeIndex, node := range anchor.PathNodes {
			if !lowerUUIDPattern.MatchString(node.ChainNodeID) {
				return fmt.Sprintf("%s.path_nodes[%d].chain_node_id", anchorPath, nodeIndex), anchor.CenterChainNodeID
			}
		}
	}
	return "", ""
}

func indexedBindingPath(path, field string) (int, bool) {
	prefix := field + "["
	if !strings.HasPrefix(path, prefix) {
		return 0, false
	}
	end := strings.IndexByte(path[len(prefix):], ']')
	if end < 0 {
		return 0, false
	}
	index, err := strconv.Atoi(path[len(prefix) : len(prefix)+end])
	return index, err == nil
}

func eventPublicationShape() *bindingShape {
	collector := objectShape(map[string]*bindingShape{"artifact_id": scalarShape, "collector_execution_id": scalarShape})
	provenance := objectShape(map[string]*bindingShape{
		"extractor_execution_id": scalarShape, "extractor_agent_version": scalarShape,
		"collector_executions": arrayShape(collector),
	})
	rawDocument := objectShape(map[string]*bindingShape{
		"artifact_id": scalarShape, "content_sha256": scalarShape, "source_ref": scalarShape,
		"source_name": scalarShape, "source_type": scalarShape, "source_url": scalarShape,
		"title": scalarShape, "published_at": scalarShape, "collected_at": scalarShape,
		"language": scalarShape, "mime_type": scalarShape,
	})
	evidence := objectShape(map[string]*bindingShape{
		"artifact_id": scalarShape, "evidence_relation": scalarShape, "evidence_excerpt": scalarShape,
		"supports_fields": arrayShape(scalarShape), "source_level": scalarShape, "is_primary": scalarShape,
	})
	tag := objectShape(map[string]*bindingShape{
		"tag_id": scalarShape, "tag_kind": scalarShape, "tag_code": scalarShape, "confidence": scalarShape,
		"assignment_reason": scalarShape, "assign_source": scalarShape,
	})
	review := objectShape(map[string]*bindingShape{
		"review_id": scalarShape, "evidence_grade": scalarShape, "reasons": arrayShape(scalarShape),
	})
	event := objectShape(map[string]*bindingShape{
		"dedupe_key": scalarShape, "title": scalarShape, "factual_summary": scalarShape,
		"occurred_at": scalarShape, "fact_payload": anyShape,
		"evidence": arrayShape(evidence), "tags": arrayShape(tag), "review": review,
	})
	return objectShape(map[string]*bindingShape{
		"package_id": scalarShape, "provenance": provenance,
		"raw_documents": arrayShape(rawDocument), "events": arrayShape(event),
	})
}

func researchThemeImportShape() *bindingShape {
	chainNode := objectShape(map[string]*bindingShape{
		"chain_node_id": scalarShape, "relation_role": scalarShape, "impact_summary": scalarShape,
	})
	event := objectShape(map[string]*bindingShape{
		"event_id": scalarShape, "evidence_role": scalarShape, "supported_claim": scalarShape,
	})
	theme := objectShape(map[string]*bindingShape{
		"theme_key": scalarShape, "name": scalarShape, "one_line_conclusion": scalarShape,
		"impact_level": scalarShape, "transmission_path": scalarShape, "trading_direction": scalarShape,
		"transmission_stage": scalarShape, "next_checkpoint": scalarShape,
		"market_confirmation_summary": scalarShape, "chain_nodes": arrayShape(chainNode), "events": arrayShape(event),
	})
	return objectShape(map[string]*bindingShape{
		"analysis_batch_id": scalarShape, "window_start": scalarShape, "window_end": scalarShape,
		"themes": arrayShape(theme),
	})
}

func researchAnchorImportShape() *bindingShape {
	event := requiredObjectShape([]string{"event_id", "evidence_role", "evidence_summary"}, map[string]*bindingShape{
		"event_id": stringShape, "evidence_role": stringShape, "evidence_summary": stringShape,
	})
	pathNode := requiredObjectShape([]string{
		"chain_node_id", "change_direction", "change_summary", "impact_summary", "incoming_transmission_mechanism",
	}, map[string]*bindingShape{
		"chain_node_id": stringShape, "change_direction": stringShape, "change_summary": stringShape,
		"impact_summary": stringShape, "incoming_transmission_mechanism": nullableStringShape,
	})
	anchor := requiredObjectShape([]string{
		"center_chain_node_id", "one_line_conclusion", "fact_summary", "net_direction_summary",
		"support_summary", "counter_summary", "trading_direction", "next_checkpoint", "events", "path_nodes",
	}, map[string]*bindingShape{
		"center_chain_node_id": stringShape, "one_line_conclusion": stringShape,
		"fact_summary": stringShape, "net_direction_summary": stringShape, "support_summary": stringShape,
		"counter_summary": nullableStringShape, "trading_direction": stringShape, "next_checkpoint": stringShape,
		"events": arrayShape(event), "path_nodes": arrayShape(pathNode),
	})
	return requiredObjectShape([]string{"theme_id", "anchors"}, map[string]*bindingShape{
		"theme_id": stringShape, "anchors": arrayShape(anchor),
	})
}
