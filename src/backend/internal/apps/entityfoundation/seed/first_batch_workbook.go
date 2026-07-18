package seed

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode"
)

const (
	firstBatchStandardSheet = "标准化保留"
	firstBatchDetailSheet   = "原名保留明细"
)

type FirstBatchWorkbookReview struct {
	Nodes    []FirstBatchNodeDraft    `json:"nodes"`
	Mappings []FirstBatchMappingDraft `json:"mappings"`
}

func LoadFirstBatchWorkbook(filePath string) (FirstBatchWorkbookReview, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return FirstBatchWorkbookReview{}, fmt.Errorf("open first-batch workbook: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return FirstBatchWorkbookReview{}, fmt.Errorf("stat first-batch workbook: %w", err)
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return FirstBatchWorkbookReview{}, fmt.Errorf("read first-batch workbook: %w", err)
	}
	reader, err := newXLSXReader(archive)
	if err != nil {
		return FirstBatchWorkbookReview{}, err
	}
	standardRows, err := reader.sheetRows(firstBatchStandardSheet)
	if err != nil {
		return FirstBatchWorkbookReview{}, err
	}
	detailRows, err := reader.sheetRows(firstBatchDetailSheet)
	if err != nil {
		return FirstBatchWorkbookReview{}, err
	}
	return parseFirstBatchRows(standardRows, detailRows)
}

func parseFirstBatchRows(standardRows, detailRows [][]string) (FirstBatchWorkbookReview, error) {
	standard, err := rowsByHeader(standardRows, []string{"标准化节点名", "原始别名", "边界较宽"})
	if err != nil {
		return FirstBatchWorkbookReview{}, fmt.Errorf("parse %s: %w", firstBatchStandardSheet, err)
	}
	details, err := rowsByHeader(detailRows, []string{"标准化节点名", "来源分类", "同花顺名称", "东方财富名称", "来源代码"})
	if err != nil {
		return FirstBatchWorkbookReview{}, fmt.Errorf("parse %s: %w", firstBatchDetailSheet, err)
	}

	review := FirstBatchWorkbookReview{}
	canonicals := map[string]struct{}{}
	for index, row := range standard {
		canonical := strings.TrimSpace(row["标准化节点名"])
		if canonical == "" {
			return FirstBatchWorkbookReview{}, fmt.Errorf("%s row %d canonical name is required", firstBatchStandardSheet, index+2)
		}
		if _, duplicate := canonicals[canonical]; duplicate {
			return FirstBatchWorkbookReview{}, fmt.Errorf("duplicate standardized canonical name %q", canonical)
		}
		canonicals[canonical] = struct{}{}
		review.Nodes = append(review.Nodes, FirstBatchNodeDraft{
			CanonicalName: canonical,
			OriginalNames: splitWorkbookList(row["原始别名"]),
			WideBoundary:  strings.TrimSpace(row["边界较宽"]) == "是",
		})
	}

	seenMappings := map[string]FirstBatchMappingDraft{}
	for index, row := range details {
		canonical := strings.TrimSpace(row["标准化节点名"])
		if _, exists := canonicals[canonical]; !exists {
			return FirstBatchWorkbookReview{}, fmt.Errorf("%s row %d references unknown canonical name %q", firstBatchDetailSheet, index+2, canonical)
		}
		taxonomy, resolved := normalizeWorkbookTaxonomy(row["来源分类"])
		for _, sourceCode := range splitWorkbookList(row["来源代码"]) {
			parts := strings.SplitN(sourceCode, ":", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
				return FirstBatchWorkbookReview{}, fmt.Errorf("%s row %d invalid source code %q", firstBatchDetailSheet, index+2, sourceCode)
			}
			sourceSystem, externalName, err := workbookSource(strings.TrimSpace(parts[0]), row)
			if err != nil {
				return FirstBatchWorkbookReview{}, fmt.Errorf("%s row %d: %w", firstBatchDetailSheet, index+2, err)
			}
			mapping := FirstBatchMappingDraft{
				CanonicalName:      canonical,
				SourceSystem:       sourceSystem,
				SourceTaxonomyType: taxonomy,
				ExternalCode:       strings.TrimSpace(parts[1]),
				ExternalName:       strings.TrimSpace(externalName),
				TaxonomyResolved:   resolved,
			}
			identity := sourceSystem + "|" + mapping.ExternalCode
			if existing, duplicate := seenMappings[identity]; duplicate {
				if existing != mapping {
					return FirstBatchWorkbookReview{}, fmt.Errorf("external code %q has conflicting workbook rows", identity)
				}
				continue
			}
			seenMappings[identity] = mapping
			review.Mappings = append(review.Mappings, mapping)
		}
	}
	return review, nil
}

func rowsByHeader(rows [][]string, required []string) ([]map[string]string, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("sheet is empty")
	}
	headers := map[string]int{}
	for index, value := range rows[0] {
		headers[strings.TrimSpace(value)] = index
	}
	for _, header := range required {
		if _, exists := headers[header]; !exists {
			return nil, fmt.Errorf("missing header %q", header)
		}
	}
	result := make([]map[string]string, 0, len(rows)-1)
	for _, values := range rows[1:] {
		row := map[string]string{}
		empty := true
		for header, index := range headers {
			if index < len(values) {
				row[header] = values[index]
				if strings.TrimSpace(values[index]) != "" {
					empty = false
				}
			}
		}
		if !empty {
			result = append(result, row)
		}
	}
	return result, nil
}

func splitWorkbookList(value string) []string {
	return normalizeOriginalNames(strings.FieldsFunc(value, func(r rune) bool {
		return r == '；' || r == ';' || r == '\n' || r == '\r'
	}))
}

func normalizeWorkbookTaxonomy(value string) (string, bool) {
	switch strings.TrimSpace(value) {
	case "行业板块":
		return ExternalTaxonomyIndustry, true
	case "概念板块":
		return ExternalTaxonomyConcept, true
	case "指数板块":
		return ExternalTaxonomyIndex, true
	default:
		return "", false
	}
}

func workbookSource(source string, row map[string]string) (string, string, error) {
	switch source {
	case "东方财富":
		if strings.TrimSpace(row["东方财富名称"]) == "" {
			return "", "", fmt.Errorf("eastmoney external name is required")
		}
		return ExternalSourceEastmoney, row["东方财富名称"], nil
	case "同花顺":
		if strings.TrimSpace(row["同花顺名称"]) == "" {
			return "", "", fmt.Errorf("ths external name is required")
		}
		return ExternalSourceTHS, row["同花顺名称"], nil
	default:
		return "", "", fmt.Errorf("unsupported source system %q", source)
	}
}

type xlsxReader struct {
	files         map[string]*zip.File
	sheetPaths    map[string]string
	sharedStrings []string
}

func newXLSXReader(archive *zip.Reader) (*xlsxReader, error) {
	reader := &xlsxReader{files: map[string]*zip.File{}, sheetPaths: map[string]string{}}
	for _, file := range archive.File {
		reader.files[path.Clean(file.Name)] = file
	}
	workbookData, err := reader.readFile("xl/workbook.xml")
	if err != nil {
		return nil, err
	}
	relationshipData, err := reader.readFile("xl/_rels/workbook.xml.rels")
	if err != nil {
		return nil, err
	}
	var workbook struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			ID   string `xml:"id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := xml.Unmarshal(workbookData, &workbook); err != nil {
		return nil, fmt.Errorf("decode workbook metadata: %w", err)
	}
	var relationships struct {
		Items []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := xml.Unmarshal(relationshipData, &relationships); err != nil {
		return nil, fmt.Errorf("decode workbook relationships: %w", err)
	}
	targets := map[string]string{}
	for _, relationship := range relationships.Items {
		target := strings.TrimPrefix(relationship.Target, "/")
		if !strings.HasPrefix(target, "xl/") {
			target = path.Join("xl", target)
		}
		targets[relationship.ID] = path.Clean(target)
	}
	for _, sheet := range workbook.Sheets {
		if target := targets[sheet.ID]; target != "" {
			reader.sheetPaths[sheet.Name] = target
		}
	}
	if _, exists := reader.files["xl/sharedStrings.xml"]; exists {
		sharedData, err := reader.readFile("xl/sharedStrings.xml")
		if err != nil {
			return nil, err
		}
		reader.sharedStrings, err = decodeSharedStrings(sharedData)
		if err != nil {
			return nil, err
		}
	}
	return reader, nil
}

func (r *xlsxReader) readFile(name string) ([]byte, error) {
	file, exists := r.files[path.Clean(name)]
	if !exists {
		return nil, fmt.Errorf("workbook entry %q is missing", name)
	}
	stream, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open workbook entry %q: %w", name, err)
	}
	defer stream.Close()
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("read workbook entry %q: %w", name, err)
	}
	return data, nil
}

func (r *xlsxReader) sheetRows(name string) ([][]string, error) {
	sheetPath := r.sheetPaths[name]
	if sheetPath == "" {
		return nil, fmt.Errorf("workbook sheet %q is missing", name)
	}
	data, err := r.readFile(sheetPath)
	if err != nil {
		return nil, err
	}
	var worksheet struct {
		Rows []struct {
			Cells []struct {
				Reference string `xml:"r,attr"`
				Type      string `xml:"t,attr"`
				Value     string `xml:"v"`
				Inline    struct {
					Text string `xml:"t"`
					Runs []struct {
						Text string `xml:"t"`
					} `xml:"r"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(data, &worksheet); err != nil {
		return nil, fmt.Errorf("decode workbook sheet %q: %w", name, err)
	}
	rows := make([][]string, 0, len(worksheet.Rows))
	for _, sourceRow := range worksheet.Rows {
		row := []string{}
		for _, cell := range sourceRow.Cells {
			column := xlsxColumnIndex(cell.Reference)
			for len(row) <= column {
				row = append(row, "")
			}
			value := cell.Value
			switch cell.Type {
			case "s":
				index, err := strconv.Atoi(strings.TrimSpace(cell.Value))
				if err != nil || index < 0 || index >= len(r.sharedStrings) {
					return nil, fmt.Errorf("sheet %q has invalid shared string index %q", name, cell.Value)
				}
				value = r.sharedStrings[index]
			case "inlineStr":
				value = cell.Inline.Text
				for _, run := range cell.Inline.Runs {
					value += run.Text
				}
			}
			row[column] = value
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func decodeSharedStrings(data []byte) ([]string, error) {
	var table struct {
		Items []struct {
			Text string `xml:"t"`
			Runs []struct {
				Text string `xml:"t"`
			} `xml:"r"`
		} `xml:"si"`
	}
	if err := xml.Unmarshal(data, &table); err != nil {
		return nil, fmt.Errorf("decode shared strings: %w", err)
	}
	result := make([]string, 0, len(table.Items))
	for _, item := range table.Items {
		value := item.Text
		for _, run := range item.Runs {
			value += run.Text
		}
		result = append(result, value)
	}
	return result, nil
}

func xlsxColumnIndex(reference string) int {
	index := 0
	found := false
	for _, character := range reference {
		if !unicode.IsLetter(character) {
			break
		}
		found = true
		index = index*26 + int(unicode.ToUpper(character)-'A'+1)
	}
	if !found {
		return 0
	}
	return index - 1
}
