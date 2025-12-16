package glow

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dector/glowx/internal/parser"
	. "github.com/dector/glowx/internal/try"
)

func fetchLogItems() []parser.LogEntryFile {
	result := []parser.LogEntryFile{}

	dirPath := "log"
	files, err := os.ReadDir(dirPath)
	Try(err)
	for _, f := range files {
		filePath := filepath.Join(dirPath, f.Name())
		fullName := f.Name()
		extension := filepath.Ext(fullName)
		if extension != ".dj" {
			continue
		}

		fileName := strings.TrimSuffix(f.Name(), extension)
		index := strings.Split(fileName, "-")[0]
		slug := strings.Split(fileName, "-")[1]

		content, err := os.ReadFile(filePath)
		Try(err)

		entryFile := parser.LogEntryFile{
			Index:    index,
			Slug:     slug,
			FileName: fileName,
		}

		err = parser.ParseLogEntry(string(content), &entryFile)
		Try(err)

		result = append(result, entryFile)
	}

	return result
}
