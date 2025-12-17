package parser

import (
	"fmt"
	"strings"
	"time"

	. "github.com/dector/glowx/internal/try"
	"github.com/dector/kdly"
	"github.com/hjson/hjson-go/v4"
)

type LogEntryFile struct {
	Index      string
	Slug       string
	FileName   string
	Meta       LogEntryMeta
	RawContent string
}

type LogEntryMeta struct {
	Title     string
	CreatedAt time.Time
	Revision  int
	Public    bool
	Tags      []string
}

func ParseLogEntry(content string, entry *LogEntryFile) error {
	contentLines := strings.Split(content, "\n")

	headerStart := -1
	for i, line := range contentLines {
		if strings.HasPrefix(line, "---") {
			headerStart = i
			break
		}
	}
	if headerStart == -1 {
		return fmt.Errorf("could not find header start")
	}

	headerEnd := -1
	for i, line := range contentLines[headerStart+1:] {
		if strings.HasPrefix(line, "---") {
			headerEnd = i + headerStart + 1
			break
		}
	}
	if headerEnd == -1 {
		return fmt.Errorf("could not find header end")
	}

	headerContent := strings.Join(contentLines[headerStart+1:headerEnd], "\n")

	meta := parseMeta(headerContent)
	rawContent := strings.TrimSpace(strings.Join(contentLines[headerEnd+1:], "\n"))

	entry.Meta = meta
	entry.RawContent = rawContent
	return nil
}

func parseMeta(text string) LogEntryMeta {
	var meta map[string]interface{}

	// Try to parse as hjson first
	err := hjson.Unmarshal([]byte(text), &meta)
	if err != nil {
		// Fallback to KDL parsing
		doc, err := kdly.Parse(text)
		Try(err)
		meta, err = kdlDocToMap(doc)
		Try(err)
	}

	return LogEntryMeta{
		Title:     meta["title"].(string),
		CreatedAt: parseDateFromHeader(meta["createdAt"].(string)),
		Revision:  int(meta["revision"].(float64)),
		Public:    meta["public"].(bool),
		Tags:      parseTagsFromHeader(meta["tags"].([]interface{})),
	}
}

func kdlDocToMap(doc *kdly.Document) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for _, node := range doc.Nodes {
		if len(node.Arguments) > 0 {
			// Simple case: node with a single argument becomes key-value
			result[node.Name] = parseKDLValue(node.Arguments[0])
		} else if len(node.Children) > 0 {
			// Node with children - handle as array for tags case
			var values []interface{}
			for _, child := range node.Children {
				if len(child.Arguments) > 0 {
					values = append(values, parseKDLValue(child.Arguments[0]))
				}
			}
			result[node.Name] = values
		}
	}
	return result, nil
}

func parseKDLValue(val kdly.Value) interface{} {
	switch val.Type {
	case kdly.ValueTypeString:
		return val.Value
	case kdly.ValueTypeNumber:
		// Try to parse as float, the existing code expects float64 for revision
		var f float64
		fmt.Sscanf(val.Value, "%f", &f)
		return f
	case kdly.ValueTypeBoolean:
		return val.Value == "true"
	case kdly.ValueTypeNull:
		return nil
	default:
		return val.Value
	}
}

func parseDateFromHeader(dateStr string) time.Time {
	time, err := time.Parse("2006-01-02 15:04", dateStr)
	Try(err)

	return time
}

func parseTagsFromHeader(tags []interface{}) []string {
	res := make([]string, len(tags))
	for i, tag := range tags {
		res[i] = tag.(string)
	}
	return res
}
