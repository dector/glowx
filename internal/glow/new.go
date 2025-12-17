package glow

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/dector/glowx/internal/try"
)

func CreateNewPost() {
	// Find the next number
	nextNumber := findNextNumber()

	// Ask user for slug
	fmt.Print("Enter slug: ")
	reader := bufio.NewReader(os.Stdin)
	slug, err := reader.ReadString('\n')
	Try(err)

	// Clean and convert slug
	slug = strings.TrimSpace(slug)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Format the number with leading zeros
	formattedNumber := fmt.Sprintf("%05d", nextNumber)

	// Create filename
	fileName := fmt.Sprintf("%s-%s.dj", formattedNumber, slug)
	filePath := filepath.Join("log", fileName)

	// Get current time
	now := time.Now()
	createdAt := now.Format("2006-01-02 15:04")

	// Create post content with metadata
	content := fmt.Sprintf(`---
title ""
createdAt "%s"
revision 1
public #false
---

`, createdAt)

	// Write the file
	err = os.WriteFile(filePath, []byte(content), 0644)
	Try(err)

	fmt.Printf("Created new post: %s\n", filePath)
}

func findNextNumber() int {
	dirPath := "log"
	files, err := os.ReadDir(dirPath)
	Try(err)

	maxNumber := 0
	for _, f := range files {
		fullName := f.Name()
		extension := filepath.Ext(fullName)
		if extension != ".dj" {
			continue
		}

		fileName := strings.TrimSuffix(fullName, extension)
		parts := strings.Split(fileName, "-")
		if len(parts) < 2 {
			continue
		}

		indexStr := parts[0]
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			continue
		}

		if index > maxNumber {
			maxNumber = index
		}
	}

	return maxNumber + 1
}
