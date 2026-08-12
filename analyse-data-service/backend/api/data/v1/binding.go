package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
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
