package glow

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dector/glowx/internal/parser"
	"github.com/dector/glowx/internal/render"
	"github.com/dector/glowx/internal/tailwind"
)

var DirOut = "out"
var DirInOutLogs = "log"
var DirInOutTag = "t"
var FileGlowMark = ".glow_build"

type BuildCtx struct {
	BuildId string
}

func Build() {
	fmt.Println("Building...")

	ctx := BuildCtx{
		BuildId: strconv.FormatUint(uint64(rand.Uint32()), 10),
	}

	tailwind.PrepareTailwind()

	prepareOutDir(DirOut)

	items := fetchLogItems()

	// render index
	indexFilePath := filepath.Join(DirOut, DirInOutLogs, "index.html")
	os.MkdirAll(filepath.Dir(indexFilePath), os.ModePerm)
	os.WriteFile(indexFilePath, render.BuildLogIndexHtmlFileContent(ctx.BuildId, items), os.ModePerm)

	// render log entries
	for _, item := range items {
		filePath := filepath.Join(DirOut, DirInOutLogs, item.Index, "index.html")
		os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		os.WriteFile(filePath, render.BuildLogEntryHtmlFileContent(ctx.BuildId, item), os.ModePerm)
	}

	// render tags

	// collect all tags to slice
	tagsToEntries := map[string][]parser.LogEntryFile{}
	for _, item := range items {
		for _, tag := range item.Meta.Tags {
			entries := tagsToEntries[tag]
			if entries == nil {
				entries = []parser.LogEntryFile{}
			}
			tagsToEntries[tag] = append(entries, item)
		}
	}

	for tag, entries := range tagsToEntries {
		tagFilePath := filepath.Join(DirOut, DirInOutLogs, DirInOutTag, tag, "index.html")
		os.MkdirAll(filepath.Dir(tagFilePath), os.ModePerm)
		os.WriteFile(tagFilePath, render.BuildLogTagHtmlFileContent(ctx.BuildId, tag, entries), os.ModePerm)
	}

	copyStatics()

	tailwind.RunTailwind()
}
