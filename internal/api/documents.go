package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/securelens/securelens/internal/events"
	"github.com/securelens/securelens/internal/store"
)

// localExtractAndClassify parses the file, runs PII classification, persists
// a DocumentClassification row, and emits document.extracted.
// It is called when the PaddleOCR docservice is unavailable.
func (s *Server) localExtractAndClassify(extractionID, tenantID, filename string, content []byte) {
	text, err := extractTextFromFile(filename, content)
	if err != nil || strings.TrimSpace(text) == "" {
		text = string(content)
	}

	raw := s.runBasicClassification(text, nil)

	entityTypes := make([]string, 0)
	seen := map[string]bool{}
	findings := make([]map[string]any, 0, len(raw))

	for _, e := range raw {
		et, _ := e["entity"].(string)
		if et == "" {
			et, _ = e["entity_type"].(string)
		}
		if et == "" {
			continue
		}
		findings = append(findings, map[string]any{
			"entity_type": et,
			"value":       e["value"],
			"confidence":  e["confidence"],
			"start":       e["start"],
			"end":         e["end"],
		})
		if !seen[et] {
			seen[et] = true
			entityTypes = append(entityTypes, et)
		}
	}

	governed := len(entityTypes) > 0
	highestLabel := ""
	for _, et := range entityTypes {
		if label, ok := entityLabelMap[et]; ok {
			if labelPriority[label] > labelPriority[highestLabel] {
				highestLabel = label
			}
		}
	}

	etJSON, _ := json.Marshal(entityTypes)
	findJSON, _ := json.Marshal(findings)

	docClass := store.DocumentClassification{
		TenantID:     tenantID,
		DocumentID:   extractionID,
		DocumentName: filename,
		EntityTypes:  store.JSON(etJSON),
		Findings:     store.JSON(findJSON),
		Governed:     governed,
		LabelApplied: highestLabel,
	}
	s.documentClassifications.Create(context.Background(), &docClass)

	events.Emit("document.extracted", map[string]any{
		"extraction_id": extractionID,
		"tenant_id":     tenantID,
		"filename":      filename,
		"entity_types":  entityTypes,
		"entity_count":  len(findings),
		"governed":      governed,
		"label_applied": highestLabel,
		"source":        "local_extraction",
	})
}

// extractTextFromFile converts uploaded file bytes into plain text based on extension.
// It uses only stdlib — no external dependencies.
func extractTextFromFile(filename string, content []byte) (string, error) {
	dot := strings.LastIndex(filename, ".")
	ext := ""
	if dot >= 0 {
		ext = strings.ToLower(filename[dot:])
	}

	switch ext {
	case ".txt", ".md", ".log", ".text":
		return string(content), nil

	case ".csv", ".tsv":
		sep := ','
		if ext == ".tsv" {
			sep = '\t'
		}
		return parseCSVText(content, sep), nil

	case ".json":
		var obj any
		if err := json.Unmarshal(content, &obj); err != nil {
			return string(content), nil // treat as plain text on parse failure
		}
		return flattenJSONToText(obj), nil

	case ".pdf":
		text := extractPDFText(content)
		if text == "" {
			return fmt.Sprintf("[PDF: %s — %d bytes, text layer not found]", filename, len(content)), nil
		}
		return text, nil

	case ".xlsx":
		return extractXLSXText(content), nil

	case ".xls":
		// Old binary XLS — attempt UTF-8 salvage; real XLS would need a full parser
		if utf8.Valid(content) {
			return string(content), nil
		}
		return fmt.Sprintf("[XLS binary: %s — %d bytes]", filename, len(content)), nil

	case ".docx":
		return extractDOCXText(content), nil

	default:
		if utf8.Valid(content) {
			return string(content), nil
		}
		return fmt.Sprintf("[Binary file: %s — %d bytes]", filename, len(content)), nil
	}
}

// parseCSVText reads all CSV cells and joins them into space-separated lines.
func parseCSVText(content []byte, sep rune) string {
	r := csv.NewReader(bytes.NewReader(content))
	r.Comma = sep
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		// Partial failure — return what we have plus raw remainder
		return string(content)
	}
	var sb strings.Builder
	for _, row := range records {
		sb.WriteString(strings.Join(row, " "))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// extractPDFText extracts readable text from PDF byte streams.
// It targets text between BT…ET markers using Tj / TJ operators.
// This covers text-layer PDFs; scanned-image PDFs will return empty.
func extractPDFText(content []byte) string {
	// Match (text)Tj  or  [(text)...]TJ
	tjRe := regexp.MustCompile(`\(([^)\\]*(?:\\.[^)\\]*)*)\)\s*Tj`)
	tjBracketRe := regexp.MustCompile(`\[([^\]]*)\]\s*TJ`)
	literalRe := regexp.MustCompile(`\(([^)\\]*(?:\\.[^)\\]*)*)\)`)

	var parts []string

	for _, m := range tjRe.FindAllSubmatch(content, -1) {
		if s := pdfUnescape(m[1]); s != "" {
			parts = append(parts, s)
		}
	}

	for _, m := range tjBracketRe.FindAllSubmatch(content, -1) {
		for _, lit := range literalRe.FindAllSubmatch(m[1], -1) {
			if s := pdfUnescape(lit[1]); s != "" {
				parts = append(parts, s)
			}
		}
	}

	return strings.Join(parts, " ")
}

// pdfUnescape handles basic PDF string escape sequences.
func pdfUnescape(b []byte) string {
	s := string(b)
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\r`, "\r")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\\`, `\`)
	s = strings.ReplaceAll(s, `\(`, "(")
	s = strings.ReplaceAll(s, `\)`, ")")
	return strings.TrimSpace(s)
}

// extractXLSXText opens an XLSX (ZIP) file and extracts text from
// sharedStrings.xml and sheet*.xml files, stripping all XML tags.
func extractXLSXText(content []byte) string {
	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		// Not a valid zip — attempt raw UTF-8
		if utf8.Valid(content) {
			return string(content)
		}
		return ""
	}

	tagRe := regexp.MustCompile(`<[^>]+>`)

	var sb strings.Builder
	for _, f := range r.File {
		name := strings.ToLower(f.Name)
		if !strings.Contains(name, "sharedstrings") && !strings.Contains(name, "sheet") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, readErr := io.ReadAll(rc)
		rc.Close()
		if readErr != nil {
			continue
		}
		stripped := tagRe.ReplaceAll(data, []byte(" "))
		// Collapse whitespace runs
		wsRe := regexp.MustCompile(`\s+`)
		clean := wsRe.ReplaceAll(stripped, []byte(" "))
		sb.Write(bytes.TrimSpace(clean))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// extractDOCXText opens a DOCX (ZIP) file and extracts text from word/document.xml.
func extractDOCXText(content []byte) string {
	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		if utf8.Valid(content) {
			return string(content)
		}
		return ""
	}

	tagRe := regexp.MustCompile(`<[^>]+>`)
	wsRe := regexp.MustCompile(`\s+`)

	var sb strings.Builder
	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			break
		}
		data, readErr := io.ReadAll(rc)
		rc.Close()
		if readErr != nil {
			break
		}
		stripped := tagRe.ReplaceAll(data, []byte(" "))
		clean := wsRe.ReplaceAll(stripped, []byte(" "))
		sb.Write(bytes.TrimSpace(clean))
	}
	return sb.String()
}

// flattenJSONToText recursively collects all string and scalar values
// from a JSON structure and joins them as space-separated text.
func flattenJSONToText(v any) string {
	var parts []string
	collectJSONStrings(v, &parts)
	return strings.Join(parts, " ")
}

func collectJSONStrings(v any, out *[]string) {
	switch val := v.(type) {
	case string:
		if s := strings.TrimSpace(val); s != "" {
			*out = append(*out, s)
		}
	case float64:
		*out = append(*out, fmt.Sprintf("%v", val))
	case bool:
		if val {
			*out = append(*out, "true")
		} else {
			*out = append(*out, "false")
		}
	case map[string]any:
		for _, child := range val {
			collectJSONStrings(child, out)
		}
	case []any:
		for _, item := range val {
			collectJSONStrings(item, out)
		}
	}
}
