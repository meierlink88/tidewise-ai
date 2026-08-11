package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type bindingShape struct {
	fields   map[string]*bindingShape
	item     *bindingShape
	any      bool
	string   bool
	boolean  bool
	integer  bool
	null     bool
	required []string
}

var scalarShape = &bindingShape{}
var anyShape = &bindingShape{any: true}
var stringShape = &bindingShape{string: true}
var booleanShape = &bindingShape{boolean: true}
var integerShape = &bindingShape{integer: true}
var nullableStringShape = &bindingShape{string: true, null: true}

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
	var discriminator struct {
		PublicationMode string `json:"publication_mode"`
	}
	_ = json.Unmarshal(payload, &discriminator)
	if discriminator.PublicationMode == "analyst_snapshot" {
		snapshot := new(ResearchThemeSnapshotImportRequest)
		if err := decodeStrictBinding(payload, researchThemeSnapshotImportShape(), snapshot); err != nil {
			return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for the Research Theme Aggregate V3 analyst_snapshot contract", map[string]any{
				"theme_key": researchThemeKeyAtPath(payload, bindingErrorPath(err)),
				"path":      bindingErrorPath(err),
			})
		}
		return &ResearchThemeImportRequest{PublicationMode: discriminator.PublicationMode, Snapshot: snapshot}, nil
	}
	request := new(ResearchThemeImportRequest)
	if err := decodeStrictBinding(payload, researchThemeImportShape(), request); err != nil {
		path := bindingErrorPath(err)
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for the Research Theme Aggregate V2 contract", map[string]any{
			"theme_key": researchThemeKeyAtPath(payload, path),
			"path":      path,
		})
	}
	return request, nil
}

func researchThemeSnapshotImportShape() *bindingShape {
	evidenceIDs := arrayShape(stringShape)
	impact := requiredObjectShape([]string{
		"node_key", "display_name", "relation_role", "impact_direction", "impact_summary", "display_order",
	}, map[string]*bindingShape{
		"node_key": stringShape, "display_name": stringShape, "relation_role": stringShape,
		"impact_direction": stringShape, "impact_summary": nullableStringShape, "display_order": scalarShape,
	})
	themeEvent := requiredObjectShape([]string{"event_id", "evidence_role", "supported_claim"}, map[string]*bindingShape{
		"event_id": stringShape, "evidence_ids": evidenceIDs, "evidence_role": stringShape,
		"supported_claim": nullableStringShape,
	})
	theme := requiredObjectShape([]string{
		"theme_key", "title", "one_line_conclusion", "conclusion_direction", "impact_strength",
		"attention_level", "conclusion_status", "transmission_stage", "investment_guidance_action",
		"investment_guidance_summary", "time_horizon_category", "time_horizon_summary",
		"transmission_summary", "checkpoint_summary", "risk_summary", "impacts", "events",
	}, map[string]*bindingShape{
		"theme_key": stringShape, "title": stringShape, "one_line_conclusion": stringShape,
		"conclusion_direction": stringShape, "impact_strength": stringShape,
		"attention_level": nullableStringShape, "conclusion_status": nullableStringShape,
		"transmission_stage": stringShape, "investment_guidance_action": stringShape,
		"investment_guidance_summary": stringShape, "time_horizon_category": stringShape,
		"time_horizon_summary": nullableStringShape, "transmission_summary": nullableStringShape,
		"checkpoint_summary": nullableStringShape, "risk_summary": nullableStringShape,
		"impacts": arrayShape(impact), "events": arrayShape(themeEvent),
	})
	checkpoint := requiredObjectShape([]string{"type", "summary"}, map[string]*bindingShape{
		"type": stringShape, "summary": stringShape,
	})
	treeEvent := requiredObjectShape([]string{"event_id", "evidence_role", "display_order"}, map[string]*bindingShape{
		"event_id": stringShape, "evidence_ids": evidenceIDs, "evidence_role": stringShape, "display_order": scalarShape,
	})
	signal := requiredObjectShape([]string{
		"signal_key", "display_summary", "role", "display_order", "variable_name", "direction",
	}, map[string]*bindingShape{
		"signal_key": stringShape, "display_summary": stringShape, "role": stringShape,
		"display_order": scalarShape, "variable_name": nullableStringShape, "direction": nullableStringShape,
	})
	incoming := requiredObjectShape([]string{"title", "mechanism", "condition_summary"}, map[string]*bindingShape{
		"title": nullableStringShape, "mechanism": stringShape, "condition_summary": nullableStringShape,
	})
	node := requiredObjectShape([]string{
		"node_key", "display_name", "position", "state_summary", "impact_direction", "impact_strength",
		"impact_summary", "reasoning_basis_summary", "evidence_gap_summary", "incoming_transmission", "signals",
	}, map[string]*bindingShape{
		"node_key": stringShape, "display_name": stringShape, "position": scalarShape,
		"state_summary": nullableStringShape, "impact_direction": stringShape, "impact_strength": stringShape,
		"impact_summary": nullableStringShape, "reasoning_basis_summary": nullableStringShape,
		"evidence_gap_summary":  nullableStringShape,
		"incoming_transmission": &bindingShape{fields: incoming.fields, required: incoming.required, null: true},
		"signals":               arrayShape(signal),
	})
	tree := requiredObjectShape([]string{
		"tree_key", "display_name", "title", "display_order", "one_line_conclusion", "fact_summary",
		"transmission_summary", "impact_direction", "impact_strength", "impact_summary",
		"conclusion_boundary_summary", "support_summary", "counter_summary", "invalidation_conditions",
		"checkpoints", "events", "nodes",
	}, map[string]*bindingShape{
		"tree_key": stringShape, "display_name": stringShape, "title": stringShape, "display_order": scalarShape,
		"one_line_conclusion": stringShape, "fact_summary": nullableStringShape,
		"transmission_summary": nullableStringShape, "impact_direction": stringShape,
		"impact_strength": stringShape, "impact_summary": nullableStringShape,
		"conclusion_boundary_summary": nullableStringShape, "support_summary": nullableStringShape,
		"counter_summary": nullableStringShape, "invalidation_conditions": arrayShape(stringShape),
		"checkpoints": arrayShape(checkpoint), "events": arrayShape(treeEvent), "nodes": arrayShape(node),
	})
	return requiredObjectShape([]string{
		"publication_mode", "analysis_batch_id", "analysis_as_of", "discovery_window_start",
		"discovery_window_end", "theme", "reasoning_trees",
	}, map[string]*bindingShape{
		"publication_mode": stringShape, "analysis_batch_id": stringShape, "analysis_as_of": stringShape,
		"discovery_window_start": stringShape, "discovery_window_end": stringShape,
		"theme": theme, "reasoning_trees": arrayShape(tree),
	})
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
	if token == nil && shape.null {
		return nil
	}
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
		if _, ok := token.(string); !ok {
			return &bindingError{path: path, err: fmt.Errorf("must be a JSON string")}
		}
		return nil
	}
	if shape.boolean {
		if _, ok := token.(bool); !ok {
			return &bindingError{path: path, err: fmt.Errorf("must be a JSON boolean")}
		}
		return nil
	}
	if shape.integer {
		number, ok := token.(json.Number)
		if !ok {
			return &bindingError{path: path, err: fmt.Errorf("must be a JSON integer")}
		}
		if _, err := strconv.ParseInt(number.String(), 10, 64); err != nil {
			return &bindingError{path: path, err: fmt.Errorf("must be a JSON integer")}
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
		Theme struct {
			ThemeKey string `json:"theme_key"`
		} `json:"theme"`
	}
	_ = json.Unmarshal(payload, &hint)
	return hint.Theme.ThemeKey
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
		"artifact_id": scalarShape, "evidence_relation": scalarShape, "evidence_statement": scalarShape,
		"supports_fields": arrayShape(scalarShape), "source_level": scalarShape,
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
	impact := objectShape(map[string]*bindingShape{
		"chain_node_entity_id": scalarShape, "relation_role": scalarShape,
		"impact_direction": scalarShape, "impact_summary": nullableStringShape,
		"display_order": scalarShape,
	})
	event := objectShape(map[string]*bindingShape{
		"event_id": scalarShape, "evidence_role": scalarShape, "supported_claim": nullableStringShape,
	})
	theme := objectShape(map[string]*bindingShape{
		"theme_key": scalarShape, "title": scalarShape, "one_line_conclusion": scalarShape,
		"conclusion_direction": scalarShape, "impact_strength": scalarShape,
		"attention_level": nullableStringShape, "conclusion_status": nullableStringShape,
		"transmission_stage": scalarShape, "investment_guidance_action": scalarShape,
		"investment_guidance_summary": scalarShape, "time_horizon_category": scalarShape,
		"time_horizon_summary": nullableStringShape, "transmission_summary": nullableStringShape,
		"checkpoint_summary": nullableStringShape, "risk_summary": nullableStringShape,
		"impacts": arrayShape(impact), "events": arrayShape(event),
	})
	checkpoint := objectShape(map[string]*bindingShape{
		"type": stringShape, "summary": stringShape,
	})
	treeEvent := objectShape(map[string]*bindingShape{
		"event_id": stringShape, "evidence_role": stringShape, "display_order": scalarShape,
	})
	signalLineage := requiredObjectShape([]string{
		"source_kind", "variable_signal_id", "semantic_submission_id", "evidence_id",
		"evidence_hash", "upstream_variable_signal_id", "upstream_direct_impact_assertion_id",
		"entity_relation_id", "industry_chain_graph_edge_id",
	}, map[string]*bindingShape{
		"source_kind": stringShape, "variable_signal_id": nullableStringShape,
		"semantic_submission_id": nullableStringShape, "evidence_id": nullableStringShape,
		"evidence_hash": nullableStringShape, "upstream_variable_signal_id": nullableStringShape,
		"upstream_direct_impact_assertion_id": nullableStringShape,
		"entity_relation_id":                  nullableStringShape, "industry_chain_graph_edge_id": nullableStringShape,
	})
	signal := objectShape(map[string]*bindingShape{
		"variable_signal_key": stringShape, "signal_role": stringShape,
		"signal_direction": stringShape, "display_summary": stringShape, "display_order": scalarShape,
		"lineage": signalLineage,
	})
	incomingLineage := requiredObjectShape([]string{
		"source_kind", "direct_impact_assertion_id", "semantic_submission_id", "evidence_id",
		"evidence_hash", "affected_variable_key", "affected_direction",
		"upstream_variable_signal_id", "upstream_direct_impact_assertion_id", "entity_relation_id",
	}, map[string]*bindingShape{
		"source_kind": stringShape, "direct_impact_assertion_id": nullableStringShape,
		"semantic_submission_id": nullableStringShape, "evidence_id": nullableStringShape,
		"evidence_hash": nullableStringShape, "affected_variable_key": nullableStringShape,
		"affected_direction": nullableStringShape, "upstream_variable_signal_id": nullableStringShape,
		"upstream_direct_impact_assertion_id": nullableStringShape,
		"entity_relation_id":                  nullableStringShape,
	})
	node := requiredObjectShape([]string{
		"position", "chain_node_entity_id", "state_summary", "impact_direction",
		"impact_strength", "impact_summary", "reasoning_basis_summary", "evidence_gap_summary",
		"incoming_industry_chain_graph_edge_id", "incoming_transmission_title",
		"incoming_transmission_mechanism", "incoming_condition_summary", "incoming_lineage", "signals",
	}, map[string]*bindingShape{
		"position": scalarShape, "chain_node_entity_id": stringShape,
		"state_summary": nullableStringShape, "impact_direction": stringShape,
		"impact_strength": stringShape, "impact_summary": nullableStringShape,
		"reasoning_basis_summary": nullableStringShape, "evidence_gap_summary": nullableStringShape,
		"incoming_industry_chain_graph_edge_id": nullableStringShape,
		"incoming_transmission_title":           nullableStringShape,
		"incoming_transmission_mechanism":       nullableStringShape,
		"incoming_condition_summary":            nullableStringShape,
		"incoming_lineage":                      &bindingShape{fields: incomingLineage.fields, required: incomingLineage.required, null: true},
		"signals":                               arrayShape(signal),
	})
	tree := requiredObjectShape([]string{
		"industry_chain_entity_id", "title", "display_order", "one_line_conclusion",
		"fact_summary", "transmission_summary", "impact_direction", "impact_strength",
		"impact_summary", "conclusion_boundary_summary", "support_summary", "counter_summary",
		"invalidation_conditions", "checkpoints", "events", "nodes",
	}, map[string]*bindingShape{
		"industry_chain_entity_id": stringShape, "title": stringShape, "display_order": scalarShape,
		"one_line_conclusion": stringShape, "fact_summary": nullableStringShape,
		"transmission_summary": nullableStringShape, "impact_direction": stringShape,
		"impact_strength": stringShape, "impact_summary": nullableStringShape,
		"conclusion_boundary_summary": nullableStringShape, "support_summary": nullableStringShape,
		"counter_summary": nullableStringShape, "invalidation_conditions": arrayShape(stringShape),
		"checkpoints": arrayShape(checkpoint), "events": arrayShape(treeEvent), "nodes": arrayShape(node),
	})
	return requiredObjectShape([]string{
		"analysis_batch_id", "analysis_as_of", "discovery_window_start",
		"discovery_window_end", "theme", "reasoning_trees",
	}, map[string]*bindingShape{
		"analysis_batch_id": stringShape, "analysis_as_of": stringShape,
		"discovery_window_start": stringShape, "discovery_window_end": stringShape,
		"theme": theme, "reasoning_trees": arrayShape(tree),
	})
}
