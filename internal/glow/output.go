package glow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dector/glowx/internal/fs"
)

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

func copyStatics() {
	err := fs.CopyDir("log/static", "out")
	if err != nil {
		panic(fmt.Errorf("Failed to copy statics: %w", err))
	}
}
