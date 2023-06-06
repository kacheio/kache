package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// licenseDefaultFile is the name of the license file.
	licenseDefaultFile = "LICENSE"

	// licenseHeaderPrefix is the unique prefix identifying a license header.
	licenseHeaderPrefix = "// MIT License"
)

// Directories to be excluded.
var exluded = map[string]struct{}{
	"vendor":  {},
	".git":    {},
	".vscode": {},
}

// readLicense reads the license from file.
func readLicense(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// makeComment wraps the given string in a comment.
func makeComment(s, prefix string) (string, error) {
	if prefix == "" {
		prefix = "//"
	}

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			lines = append(lines, prefix+"\n")
		} else {
			lines = append(lines, fmt.Sprintf("%s %s\n", prefix, line))
		}
	}
	lines = append(lines, "\n")

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, ""), nil
}

// hasLicenseHeader check if a file has a valid license header.
func hasLicenseHeader(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return false
	}

	return strings.HasPrefix(string(data), licenseHeaderPrefix)
}

// addLicenseHeader adds a license header to the source file.
func addLicenseHeader(path string, license string, update bool) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Make the license a comment.
	header, err := makeComment(license, "//")
	if err != nil {
		return err
	}

	// Remove existing header on update.
	source := string(data)
	if update {
		source = removeLicenseHeader(source)
	}

	// Prefix source with new license header.
	content := []byte(header + source)

	// Truncate the file before writing the new content.
	err = file.Truncate(0)
	if err != nil {
		return err
	}

	// Seek to the beginning of the file.
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	// Write the new content.
	_, err = file.Write(content)
	if err != nil {
		return err
	}

	return nil
}

// removeLicenseHeader if present.
func removeLicenseHeader(s string) string {
	if !strings.HasPrefix(s, licenseHeaderPrefix) {
		return s
	}

	// Remove lines until a line without a comment prefix is found.
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "//") {
			lines = lines[i:]
			break
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func main() {
	licenseFile := flag.String("license", licenseDefaultFile, "License file")
	root := flag.String("dir", ".", "Root directory path")
	list := flag.Bool("list", false, "List all files without a license header (no update)")
	force := flag.Bool("force", false, "Force forces an update of the license header")

	flag.Parse()

	if *root == "" {
		fmt.Println("Please provide the root directory path using the -dir flag")
		return
	}

	license, err := readLicense(*licenseFile)
	if err != nil {
		fmt.Printf("Error reading license file: %v\n", err)
		return
	}

	var filesToUpdate []string
	err = filepath.Walk(*root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if _, ok := exluded[info.Name()]; ok {
				return filepath.SkipDir
			}
			return nil
		}

		// Only add license to .go files, but exclude generated ones.
		if !strings.HasSuffix(info.Name(), ".go") || strings.HasSuffix(info.Name(), ".pb.go") {
			return nil
		}

		if !hasLicenseHeader(path) || *force {
			filesToUpdate = append(filesToUpdate, path)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if *list {
		fmt.Println("Source files without license headers:")
		for _, file := range filesToUpdate {
			fmt.Println(file)
		}
		return
	}

	// Add or update license headers.
	for _, file := range filesToUpdate {
		if err := addLicenseHeader(file, license, *force); err != nil {
			fmt.Printf("Error adding license header to %s: %v\n", file, err)
		} else {
			fmt.Printf("Added license header to %s\n", file)
		}
	}
}
