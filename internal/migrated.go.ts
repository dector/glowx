// Translated by Gemini 2, but it's kinda failed.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	LogsDir           = "log"
	OutRoot           = "out"
	OutLog            = "out/log"
	IndexTemplate     = "src/templates/index.html"
	LogLayoutTemplate = "src/templates/log/layout.html"
	LogIndexTemplate  = "src/templates/log/index.html"
	LogEntryTemplate  = "src/templates/log/entry_page.html"
)

var LogTemplates = struct {
	NewLog string
}{
	NewLog: `---
title: {{.Title}}
createdAt: {{.Date}}
revision: 1
public: no
tags:
  -
---

D.\n`,
}

// type Command string

// const (
// 	HelpCommand    Command = "help"
// 	VersionCommand Command = "version"
// 	LogCommand     Command = "log"
// 	BuildCommand   Command = "build"
// )

// type SubCommand string

// type CommandContext struct {
// 	Command    Command
// 	SubCommand SubCommand
// 	Args       []string
// }

type LogItem struct {
	LogIndex       string
	LogDate        time.Time
	FormattedDate  string
	SourceFileName string
	Header         LogHeader
	ContentDjot    string // TODO: Replace with djot AST representation if needed
	ContentHtml    string
}

type LogHeader struct {
	Title     string   `yaml:"title"`
	CreatedAt string   `yaml:"createdAt"`
	Revision  int      `yaml:"revision"`
	Public    bool     `yaml:"public"`
	Tags      []string `yaml:"tags"`
}

type TemplateData struct {
	Page PageData
}

type PageData struct {
	Title    string
	BuildUid int64
	InHead   string
	Content  string
}

type LogIndexTemplateData struct {
	PageH1   string
	Items    []LogIndexItem
	ShowBack bool
}

type LogIndexItem struct {
	Title string
	Path  string
}

type LogEntryTemplateData struct {
	Content string
	Title   string
	Date    string
	Tags    []string
}

// func main() {
// 	commandContext, err := parseCommand(os.Args[1:])
// 	if err != nil {
// 		fmt.Println("Error parsing command:", err)
// 		os.Exit(1)
// 	}

// 	executeCommand(commandContext)
// }

func parseCommand(args []string) (CommandContext, error) {
	var command Command
	if len(args) > 0 {
		command = Command(args[0])
	}

	var subCommand SubCommand
	if len(args) > 1 {
		subCommand = SubCommand(args[1])
	}

	return CommandContext{
		Command:    command,
		SubCommand: subCommand,
		Args:       args,
	}, nil
}

func executeCommand(ctx CommandContext) {
	switch ctx.Command {
	case LogCommand:
		executeLogCommand(ctx.SubCommand)
	case BuildCommand:
		isDev := contains(ctx.Args, "--dev")
		executeBuildCommand(isDev)
	default:
		fmt.Println("Invalid command")
		os.Exit(1)
	}
}

func executeLogCommand(subCommand SubCommand) {
	switch subCommand {
	case "new":
		createNewLog()
	default:
		fmt.Println("Invalid log subcommand")
		os.Exit(1)
	}
}

func createNewLog() {
	latestLogIndex := 0
	err := filepath.Walk(LogsDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".dj") {
			return nil
		}

		parts := strings.Split(strings.Split(info.Name(), ".")[0], "-")
		if len(parts) > 0 {
			logIndex, err := strconv.Atoi(parts[0])
			if err == nil && logIndex > latestLogIndex {
				latestLogIndex = logIndex
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error reading log directory:", err)
		os.Exit(1)
	}

	newIndex := fmt.Sprintf("%05d", latestLogIndex+1)
	fmt.Print("Title: ")
	title, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	title = strings.TrimSpace(title)

	fmt.Print("Slug: ")
	slug, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	slug = strings.TrimSpace(slug)
	if slug == "" {
		slug = strings.ReplaceAll(strings.ToLower(title), " ", "-")
	}

	fileName := fmt.Sprintf("%s-%s.dj", newIndex, slug)
	file := filepath.Join(LogsDir, fileName)

	t := template.Must(template.New("newlog").Parse(LogTemplates.NewLog))
	var contentBuffer bytes.Buffer
	err = t.Execute(&contentBuffer, map[string]interface{}{
		"Title": title,
		"Date":  time.Now().Format("2006-01-02T15:04"),
	})
	if err != nil {
		fmt.Println("Error rendering template:", err)
		os.Exit(1)
	}

	err = os.WriteFile(file, contentBuffer.Bytes(), 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		os.Exit(1)
	}

	printResult("ok", fileName)
}

func executeBuildCommand(isDev bool) {
	items := collectLogItems()
	fmt.Printf("Found: %d items\n", len(items))

	buildUid := time.Now().Unix()

	pageGlobal := PageData{
		Title:    "/dector/log",
		BuildUid: buildUid,
		InHead: `<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Noto+Sans+Mono:wght@100..900&display=swap" rel="stylesheet">`,
	}

	err := os.RemoveAll(OutRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Println("Error removing output directory:", err)
		os.Exit(1)
	}

	err = os.MkdirAll(OutRoot, 0755)
	if err != nil {
		fmt.Println("Error creating output directory:", err)
		os.Exit(1)
	}

	err = os.MkdirAll(OutLog, 0755)
	if err != nil {
		fmt.Println("Error creating log output directory:", err)
		os.Exit(1)
	}

	renderEntryHtml := func(item LogItem) string {
		contentHtml := item.ContentHtml

		entryHtml, err := renderTemplate(LogEntryTemplate, LogEntryTemplateData{
			Content: contentHtml,
			Title:   item.Header.Title,
			Date:    item.FormattedDate,
			Tags:    item.Header.Tags,
		})
		if err != nil {
			fmt.Println("Error rendering entry template:", err)
			os.Exit(1)
		}

		pageHtml, err := renderTemplate(LogLayoutTemplate, TemplateData{
			Page: PageData{
				Title:    fmt.Sprintf("%s ~ %s", pageGlobal.Title, item.Header.Title),
				BuildUid: pageGlobal.BuildUid,
				InHead:   pageGlobal.InHead,
				Content:  entryHtml,
			},
		})
		if err != nil {
			fmt.Println("Error rendering layout template:", err)
			os.Exit(1)
		}

		return pageHtml
	}

	renderListHtml := func(items []LogItem, title string, showBack bool) string {
		logIndexItems := make([]LogIndexItem, len(items))
		for i, item := range items {
			logIndexItems[i] = LogIndexItem{
				Title: item.Header.Title,
				Path:  fmt.Sprintf("/log/%s", item.LogIndex),
			}
		}

		entryHtml, err := renderTemplate(LogIndexTemplate, LogIndexTemplateData{
			PageH1:   title,
			Items:    logIndexItems,
			ShowBack: showBack,
		})

		if err != nil {
			fmt.Println("Error rendering index template:", err)
			os.Exit(1)
		}

		pageHtml, err := renderTemplate(LogLayoutTemplate, TemplateData{
			Page: PageData{
				Title:    pageGlobal.Title,
				BuildUid: pageGlobal.BuildUid,
				InHead:   pageGlobal.InHead,
				Content:  entryHtml,
			},
		})
		if err != nil {
			fmt.Println("Error rendering layout template:", err)
			os.Exit(1)
		}

		return pageHtml
	}

	for _, item := range items {
		pageHtml := renderEntryHtml(item)

		itemDir := filepath.Join(OutLog, item.LogIndex)
		err = os.MkdirAll(itemDir, 0755)
		if err != nil {
			fmt.Println("Error creating item directory:", err)
			os.Exit(1)
		}

		file := filepath.Join(itemDir, "index.html")
		err = os.WriteFile(file, []byte(pageHtml), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			os.Exit(1)
		}
	}

	{
		pageHtml := renderListHtml(items, "/log", false)
		file := filepath.Join(OutLog, "index.html")
		err = os.WriteFile(file, []byte(pageHtml), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			os.Exit(1)
		}
	}

	{
		html, err := renderTemplate(IndexTemplate, TemplateData{
			Page: pageGlobal,
		})
		if err != nil {
			fmt.Println("Error rendering index template:", err)
			os.Exit(1)
		}

		file := filepath.Join(OutRoot, "index.html")
		err = os.WriteFile(file, []byte(html), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			os.Exit(1)
		}
	}

	// Render tags
	{
		tags := collectTags(items)
		for _, tag := range tags {
			itemsWithTag := filter(items, func(item LogItem) bool {
				return contains(item.Header.Tags, tag)
			})

			pageHtml := renderListHtml(itemsWithTag, fmt.Sprintf("t/%s", tag), true)

			tagDir := filepath.Join(OutLog, "t", tag)
			err = os.MkdirAll(tagDir, 0755)
			if err != nil {
				fmt.Println("Error creating tag directory:", err)
				os.Exit(1)
			}

			file := filepath.Join(tagDir, "index.html")
			err = os.WriteFile(file, []byte(pageHtml), 0644)
			if err != nil {
				fmt.Println("Error writing file:", err)
				os.Exit(1)
			}
		}
	}

	printResult("ok", "")
}

func collectLogItems() []LogItem {
	var items []LogItem

	err := filepath.Walk(LogsDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".dj") {
			return nil
		}

		sourceFileName := info.Name()
		fmt.Printf("Parsing %s\n", sourceFileName)

		file, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(file)
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

		var header LogHeader
		err = yaml.Unmarshal([]byte(headerContent), &header)
		if err != nil {
			return fmt.Errorf("error unmarshaling YAML: %w", err)
		}

		djotContent := strings.TrimSpace(strings.Join(contentLines[headerEnd+1:], "\n"))

		parsedContent, err := parseDjot(djotContent)
		if err != nil {
			return fmt.Errorf("error parsing djot: %w", err)
		}

		renderedContent, err := renderHTML(parsedContent)
		if err != nil {
			return fmt.Errorf("error rendering HTML: %w", err)
		}

		logNameProps := strings.Split(sourceFileName[:len(sourceFileName)-len(filepath.Ext(sourceFileName))], "-")
		logIndex := logNameProps[0]

		logDate, err := time.Parse("2006-01-02T15:04", header.CreatedAt)
		if err != nil {
			return fmt.Errorf("error parsing date: %w", err)
		}

		items = append(items, LogItem{
			LogIndex:       logIndex,
			LogDate:        logDate,
			FormattedDate:  logDate.Format("Mon, 02 Jan 2006 15:04"),
			SourceFileName: sourceFileName,
			Header:         header,
			ContentDjot:    parsedContent,
			ContentHtml:    renderedContent,
		})
		return nil
	})

	if err != nil {
		fmt.Println("Error walking log directory:", err)
		os.Exit(1)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[j].LogDate.Before(items[i].LogDate)
	})

	return items
}

func collectTags(items []LogItem) []string {
	tagsMap := make(map[string]bool)
	for _, item := range items {
		for _, tag := range item.Header.Tags {
			tagsMap[tag] = true
		}
	}

	var tags []string
	for tag := range tagsMap {
		tags = append(tags, tag)
	}

	return tags
}

func printResult(status string, message string) {
	fmt.Println()
	fmt.Printf("\033[32m%s\033[0m\n", strings.ToUpper(status))

	if message != "" {
		fmt.Printf("\033[33m%s\033[0m\n", message)
	}
}

func renderTemplate(templatePath string, data interface{}) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(content))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func filter(items []LogItem, predicate func(LogItem) bool) []LogItem {
	var result []LogItem
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

func parseDjot(djotContent string) (string, error) {
	// Execute the djot command
	cmd := exec.Command("djot", "--to", "html")
	cmd.Stdin = strings.NewReader(djotContent)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running djot: %w, stderr: %s", err, errOut.String())
	}

	return out.String(), nil
}

func renderHTML(djotHTML string) (string, error) {
	re := regexp.MustCompile(`<code class="language-([^"]+)">([\s\S]*?)</code>`)
	matches := re.FindAllStringSubmatch(djotHTML, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		language := match[1]
		code := match[2]

		var highlightedCode string
		var err error

		switch language {
		case "ts", "typescript":
			highlightedCode, err = highlightCode(code, "typescript")
			if err != nil {
				log.Println("Error highlighting typescript code:", err)
				highlightedCode = html.EscapeString(code) // Fallback to escaping
			}
		default:
			highlightedCode = html.EscapeString(code) // Default to escaping
		}

		langTitle := fmt.Sprintf(`<span class="lang-tag">%s</span>`, language)
		replacement := fmt.Sprintf(`<div class="code-block"><pre><code class="hljs" data-language="%s">%s</code></pre>%s</div>`, language, highlightedCode, langTitle)

		djotHTML = strings.Replace(djotHTML, match[0], replacement, 1)
	}

	return djotHTML, nil
}

func highlightCode(code string, language string) (string, error) {
	cmd := exec.Command("hljs", "-l", language)
	cmd.Stdin = strings.NewReader(code)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running hljs: %w, stderr: %s", err, errOut.String())
	}

	return out.String(), nil
}
