package seed

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSplitWorkbookListSortsAndDeduplicatesAliases(t *testing.T) {
	left := splitWorkbookList(" 增材制造；3D打印；增材制造；航空发动机 ")
	right := splitWorkbookList("航空发动机；增材制造；3D打印")
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("normalized aliases differ by workbook order: %v != %v", left, right)
	}
}

func TestLoadFirstBatchWorkbookSplitsAliasesAndExternalIdentifiers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "first-batch.xlsx")
	standardHeaders := []string{"标准化节点名", "保留类型", "原始别名", "合并原名数", "标准小类映射数", "边界较宽", "GB/T主要候选映射（最多5项）", "ISIC主要对应（最多3项）", "NAICS 2022候选对应（最多3项）", "保留依据", "来源平台", "来源代码"}
	detailHeaders := []string{"原始序号", "原始板块名称", "标准化节点名", "保留类型", "保留依据", "标准映射数", "边界较宽", "GB/T候选映射", "来源平台", "来源分类", "平台数", "同花顺名称", "东方财富名称", "来源代码"}
	writeWorkbookFixture(t, path, map[string][][]string{
		"标准化保留": {
			standardHeaders,
			{"3D打印", "稳定技术/工艺", "3D打印；增材制造", "2", "0", "是", "", "", "", "技术类别", "东方财富；同花顺", "东方财富:BK0619；同花顺:300127"},
			{"白酒", "稳定产品", "白酒；白酒概念", "2", "0", "否", "", "", "", "产品类别", "东方财富；同花顺", "东方财富:BK0896；同花顺:881273"},
		},
		"原名保留明细": {
			detailHeaders,
			{"1", "3D打印", "3D打印", "稳定技术/工艺", "技术类别", "0", "是", "", "东方财富、同花顺", "概念板块", "2", "3D打印", "3D打印", "东方财富:BK0619；同花顺:300127"},
			{"2", "白酒", "白酒", "稳定产品", "产品类别", "0", "否", "", "东方财富、同花顺", "概念板块、行业板块", "2", "白酒", "白酒", "东方财富:BK0896；同花顺:881273"},
		},
	})

	review, err := LoadFirstBatchWorkbook(path)
	if err != nil {
		t.Fatalf("LoadFirstBatchWorkbook() error = %v", err)
	}
	if len(review.Nodes) != 2 || len(review.Mappings) != 4 {
		t.Fatalf("review counts = nodes %d mappings %d", len(review.Nodes), len(review.Mappings))
	}
	if got := review.Nodes[0].OriginalNames; len(got) != 2 || got[1] != "增材制造" || !review.Nodes[0].WideBoundary {
		t.Fatalf("first node = %+v", review.Nodes[0])
	}
	resolved := 0
	unresolved := 0
	for _, mapping := range review.Mappings {
		if mapping.TaxonomyResolved {
			resolved++
		} else {
			unresolved++
		}
	}
	if resolved != 2 || unresolved != 2 {
		t.Fatalf("taxonomy states = resolved %d unresolved %d", resolved, unresolved)
	}
	if review.Mappings[0].SourceSystem != "eastmoney" || review.Mappings[0].ExternalName != "3D打印" || review.Mappings[0].SourceTaxonomyType != "concept_sector" {
		t.Fatalf("first mapping = %+v", review.Mappings[0])
	}
}

func writeWorkbookFixture(t *testing.T, path string, sheets map[string][][]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	orderedNames := []string{"标准化保留", "原名保留明细"}
	workbook := `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`
	rels := `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`
	for i, name := range orderedNames {
		workbook += fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlText(name), i+1, i+1)
		rels += fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i+1, i+1)
		entry, err := writer.Create(fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(sheetXML(sheets[name]))); err != nil {
			t.Fatal(err)
		}
	}
	workbook += `</sheets></workbook>`
	rels += `</Relationships>`
	for name, data := range map[string]string{"xl/workbook.xml": workbook, "xl/_rels/workbook.xml.rels": rels} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func sheetXML(rows [][]string) string {
	result := `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`
	for rowIndex, row := range rows {
		result += fmt.Sprintf(`<row r="%d">`, rowIndex+1)
		for columnIndex, value := range row {
			result += fmt.Sprintf(`<c r="%s%d" t="inlineStr"><is><t>%s</t></is></c>`, columnName(columnIndex), rowIndex+1, xmlText(value))
		}
		result += `</row>`
	}
	return result + `</sheetData></worksheet>`
}

func columnName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+index%26)) + name
		index = index/26 - 1
	}
	return name
}

func xmlText(value string) string {
	var buffer bytes.Buffer
	if err := xml.EscapeText(&buffer, []byte(value)); err != nil {
		panic(err)
	}
	return buffer.String()
}
