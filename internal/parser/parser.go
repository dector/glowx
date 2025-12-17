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
	nodeCounts := make(map[string]int)

	// First pass: count how many times each node name appears
	for _, node := range doc.Nodes {
		nodeCounts[node.Name]++
	}

	// Track which nodes have been collected as arrays
	collected := make(map[string]bool)

	for _, node := range doc.Nodes {
		// If this node name appears multiple times, collect all as array
		if nodeCounts[node.Name] > 1 && !collected[node.Name] {
			var values []interface{}
			for _, n := range doc.Nodes {
				if n.Name == node.Name && len(n.Arguments) > 0 {
					values = append(values, parseKDLValue(n.Arguments[0]))
				}
			}
			result[node.Name+"s"] = values // Use plural form (e.g., "tag" -> "tags")
			collected[node.Name] = true
		} else if !collected[node.Name] {
			// Single occurrence
			if len(node.Arguments) > 0 {
				result[node.Name] = parseKDLValue(node.Arguments[0])
			} else if len(node.Children) > 0 {
				var values []interface{}
				for _, child := range node.Children {
					if len(child.Arguments) > 0 {
						values = append(values, parseKDLValue(child.Arguments[0]))
					}
				}
				result[node.Name] = values
			}
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
