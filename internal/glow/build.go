package glow

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/hjson/hjson-go/v4"
	"github.com/sivukhin/godjot/djot_parser"
	"github.com/sivukhin/godjot/html_writer"
)

//go:embed all:assets/*
var assets embed.FS

type LogEntryFile struct {
	index      string
	slug       string
	fileName   string
	meta       LogEntryMeta
	rawContent string
}

type LogEntryMeta struct {
	title     string
	createdAt time.Time
	revision  int
	public    bool
	tags      []string
}

var DirOut = "out"
var DirInOutLogs = "log"
var DirInOutTag = "t"
var FileGlowMark = ".glow_build"

func Try(err error) {
	if err != nil {
		panic(err)
	}
}

type BuildCtx struct {
	BuildId string
}

func Build() {
	fmt.Println("Building...")

	ctx := BuildCtx{
		BuildId: strconv.FormatUint(uint64(rand.Uint32()), 10),
	}

	prepareTailwind()

	prepareOutDir(DirOut)

	items := fetchLogItems()

	// render index
	indexFilePath := filepath.Join(DirOut, DirInOutLogs, "index.html")
	os.MkdirAll(filepath.Dir(indexFilePath), os.ModePerm)
	os.WriteFile(indexFilePath, buildLogIndexHtmlFileContent(ctx, items), os.ModePerm)

	// render log entries
	for _, item := range items {
		filePath := filepath.Join(DirOut, DirInOutLogs, item.index, "index.html")
		os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		os.WriteFile(filePath, buildLogEntryHtmlFileContent(ctx, item), os.ModePerm)
	}

	// render tags

	// collect all tags to slice
	tagsToEntries := map[string][]LogEntryFile{}
	for _, item := range items {
		for _, tag := range item.meta.tags {
			entries := tagsToEntries[tag]
			if entries == nil {
				entries = []LogEntryFile{}
			}
			tagsToEntries[tag] = append(entries, item)
		}
	}

	for tag, entries := range tagsToEntries {
		tagFilePath := filepath.Join(DirOut, DirInOutLogs, DirInOutTag, tag, "index.html")
		os.MkdirAll(filepath.Dir(tagFilePath), os.ModePerm)
		os.WriteFile(tagFilePath, buildLogTagHtmlFileContent(ctx, tag, entries), os.ModePerm)
	}

	copyAssets()

	runTailwind()
}

func fetchLogItems() []LogEntryFile {
	result := []LogEntryFile{}

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

		entryFile := LogEntryFile{
			index:    index,
			slug:     slug,
			fileName: fileName,
		}

		err = parseLogEntry(string(content), &entryFile)
		Try(err)

		result = append(result, entryFile)
	}

	return result
}

func renderDjot(content string) string {
	ast := djot_parser.BuildDjotAst([]byte(content))
	html := djot_parser.NewConversionContext(
		"html",
		djot_parser.DefaultConversionRegistry,
		map[djot_parser.DjotNode]djot_parser.Conversion{},
	).ConvertDjotToHtml(&html_writer.HtmlWriter{}, ast[0])
	return html
}

func prepareOutDir(dir string) {
	fmt.Printf("Preparing %s dir...\n", dir)

	// check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, os.ModePerm)
		os.Create(filepath.Join(dir, FileGlowMark))
	} else {
		glowMark := filepath.Join(dir, FileGlowMark)
		_, err := os.Stat(glowMark)
		if os.IsNotExist(err) {
			fmt.Printf("Out dir '%s' exists but doesn't have Glow mark", dir)
			os.Exit(1)
		}

		files, _ := os.ReadDir(dir)
		for _, f := range files {
			if f.Name() == FileGlowMark {
				continue
			}
			os.RemoveAll(filepath.Join(dir, f.Name()))
		}
	}
}

func toSortedReverse(_entries []LogEntryFile) []LogEntryFile {
	entries := make([]LogEntryFile, len(_entries))
	copy(entries, _entries)

	slices.SortFunc(entries, func(a, b LogEntryFile) int {
		return -a.meta.createdAt.Compare(b.meta.createdAt)
	})

	return entries
}

func buildLogIndexHtmlFileContent(ctx BuildCtx, _entries []LogEntryFile) []byte {
	entries := toSortedReverse(_entries)

	pEntries := []struct {
		Url   string
		Title string
	}{}
	for _, entry := range entries {
		pEntries = append(pEntries, struct {
			Url   string
			Title string
		}{
			Url:   "/log/" + entry.index,
			Title: entry.meta.title,
		})
	}

	params := map[string]interface{}{
		"Page": struct {
			Title   string
			BuildId string
		}{
			Title:   "/dector/log",
			BuildId: ctx.BuildId,
		},
		"Title":   "/log",
		"Entries": pEntries,
	}

	return []byte(renderPageTemplate("log_list", params))
}

func buildLogEntryHtmlFileContent(ctx BuildCtx, entryFile LogEntryFile) []byte {
	htmlContent := renderDjot(entryFile.rawContent)

	meta := entryFile.meta
	params := map[string]interface{}{
		"Page": struct {
			Title   string
			BuildId string
		}{
			Title:   "/dector/log ~ " + meta.title,
			BuildId: ctx.BuildId,
		},
		"Entry": struct {
			Title   string
			Content string
			Date    string
			Tags    []string
		}{
			Title:   meta.title,
			Content: htmlContent,
			Date:    meta.createdAt.Format("Mon, 02 Jan 2006 15:04"),
			Tags:    meta.tags,
		},
	}

	return []byte(renderPageTemplate("log_entry", params))
}

func buildLogTagHtmlFileContent(ctx BuildCtx, tag string, _entries []LogEntryFile) []byte {
	entries := toSortedReverse(_entries)

	pEntries := []struct {
		Url   string
		Title string
	}{}
	for _, entry := range entries {
		pEntries = append(pEntries, struct {
			Url   string
			Title string
		}{
			Url:   "/log/" + entry.index,
			Title: entry.meta.title,
		})
	}

	params := map[string]interface{}{
		"Page": struct {
			Title   string
			BuildId string
		}{
			Title:   "/dector/log ~ " + tag,
			BuildId: ctx.BuildId,
		},
		"Title":    "#" + tag,
		"Entries":  pEntries,
		"ShowBack": true,
	}

	return []byte(renderPageTemplate("log_list", params))
}

func renderPageTemplate(name string, params map[string]interface{}) string {
	// Render content
	func() {
		var out bytes.Buffer
		assetFile, err := assets.ReadFile("assets/templates/" + name + ".html")
		Try(err)
		tmpl, err := template.New(name).Parse(string(assetFile))
		Try(err)

		err = tmpl.ExecuteTemplate(&out, name, params)
		Try(err)
		params["Content"] = out.String()
		params["BodyClass"] = "p-log"
	}()

	// Render HTML page
	var out bytes.Buffer
	func() {
		assetFile, err := assets.ReadFile("assets/templates/__page.html")
		Try(err)
		tmpl, err := template.New(name).Parse(string(assetFile))
		Try(err)

		err = tmpl.ExecuteTemplate(&out, name, params)
		Try(err)
	}()

	return out.String()
}

func parseLogEntry(content string, entry *LogEntryFile) error {
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

	entry.meta = meta
	entry.rawContent = rawContent
	return nil
}

func parseMeta(text string) LogEntryMeta {
	var meta map[string]interface{}
	err := hjson.Unmarshal([]byte(text), &meta)
	Try(err)

	return LogEntryMeta{
		title:     meta["title"].(string),
		createdAt: parseDateFromHeader(meta["createdAt"].(string)),
		revision:  int(meta["revision"].(float64)),
		public:    meta["public"].(bool),
		tags:      parseTagsFromHeader(meta["tags"].([]interface{})),
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

const (
	tailwindBinary  = "tailwindcss-linux-x64"
	tailwindVersion = "v4.0.12"
)

var tailwindBinaryPath = fmt.Sprintf("/tmp/glowx/%s", tailwindBinary)

func prepareTailwind() {
	_, err := os.Stat(tailwindBinaryPath)
	if os.IsNotExist(err) {
		fmt.Println("Downloading Tailwind...")

		remoteBinaryUrl := fmt.Sprintf("https://github.com/tailwindlabs/tailwindcss/releases/download/%s/%s", tailwindVersion, tailwindBinary)
		r, err := http.Get(remoteBinaryUrl)
		if err != nil {
			panic(fmt.Errorf("Failed to download: %w", err))
		}

		// Create parent dir
		err = os.MkdirAll(filepath.Dir(tailwindBinaryPath), 0755)
		if err != nil {
			panic(fmt.Errorf("Failed to create parent dir: %w", err))
		}

		file, err := os.Create(tailwindBinaryPath)
		if err != nil {
			panic(fmt.Errorf("Failed to create file: %w", err))
		}
		defer file.Close()
		Try(file.Chmod(0755))

		_, err = io.Copy(file, r.Body)
		if err != nil {
			panic(fmt.Errorf("Failed to write file: %w", err))
		}
	}
}

func runTailwind() {
	fmt.Println("Running Tailwind...")
	path, err := exec.LookPath(tailwindBinaryPath)
	if err != nil {
		panic(fmt.Errorf("Failed to find tailwind binary: %w", err))
	}

	cmd := exec.Command(path, "-i", "main.css", "-o", "out/assets/styles.css", "--minify")
	Try(err)

	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	Try(err)
}

func copyAssets() {
	// copy log/favicon.ico to out
	err := copyFile("log/favicon.ico", "out/favicon.ico")
	if err != nil {
		panic(fmt.Errorf("Failed to copy favicon: %w", err))
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Close()
}
