package parser

import (
	"fmt"
	"strings"
	"time"

	. "github.com/dector/glowx/internal/try"
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
	err := hjson.Unmarshal([]byte(text), &meta)
	Try(err)

	return LogEntryMeta{
		Title:     meta["title"].(string),
		CreatedAt: parseDateFromHeader(meta["createdAt"].(string)),
		Revision:  int(meta["revision"].(float64)),
		Public:    meta["public"].(bool),
		Tags:      parseTagsFromHeader(meta["tags"].([]interface{})),
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
