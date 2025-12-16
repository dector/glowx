package tailwind

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/dector/glowx/internal/try"
)

const (
	tailwindBinary  = "tailwindcss-linux-x64"
	tailwindVersion = "v4.0.12"
)

var tailwindBinaryPath = fmt.Sprintf("/tmp/glowx/%s", tailwindBinary)

func PrepareTailwind() {
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

func RunTailwind() {
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
