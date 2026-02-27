package main

import (
	_ "embed"

	"bytes"
	"debug/buildinfo"
	"encoding/base64"
	"fmt"
	"html"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func mainInternal() error {
	//nolint:wrapcheck
	return NewApp().Execute()
}

func main() {
	err := mainInternal()
	if err != nil {
		log.Fatal(err)
	}
}

func NewApp() *cobra.Command {
	c := cobra.Command{
		Use:   "foolhtml",
		Short: "foolhtml",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args)
		},
	}

	c.AddCommand(
		NewVersionCommand(),
	)

	return &c
}

// FileContent holds the name and escaped HTML content of a file.
type FileContent struct {
	Name       string
	RawContent string
	JSSafeHTML string
}

// Data for the HTML template
type TemplateData struct {
	Files []FileContent
}

//go:embed template.html.tpl
var mainTemplate string

// linkRegex matches <link rel="stylesheet" href="...">
var linkRegex = regexp.MustCompile(`(?i)<link[^>]+rel=["']stylesheet["'][^>]+href=["']([^"']+)["'][^>]*>`)

// scriptRegex matches <script src="..."></script>
var scriptRegex = regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']+)["'][^>]*>\s*</script>`)

// imgRegex matches <img src="...">
var imgRegex = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["'][^>]*>`)

func isRemote(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "//")
}

func inlineResources(htmlPath string, content string) string {
	baseDir := filepath.Dir(htmlPath)

	// 1. Inline CSS
	content = linkRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatch := linkRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		href := submatch[1]
		if isRemote(href) {
			return match
		}
		target := filepath.Join(baseDir, href)

		resContent, err := os.ReadFile(target)
		if err != nil {
			log.Printf("Warning: Failed to fetch CSS resource %s: %v\n", target, err)
			return match
		}
		return fmt.Sprintf("<style>%s</style>", string(resContent))
	})

	// 2. Inline JavaScript
	content = scriptRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatch := scriptRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		src := submatch[1]
		if isRemote(src) {
			return match
		}
		target := filepath.Join(baseDir, src)

		resContent, err := os.ReadFile(target)
		if err != nil {
			log.Printf("Warning: Failed to fetch JS resource %s: %v\n", target, err)
			return match
		}
		return fmt.Sprintf("<script>%s</script>", string(resContent))
	})

	// 3. Inline Images
	content = imgRegex.ReplaceAllStringFunc(content, func(match string) string {
		srcRegex := regexp.MustCompile(`(?i)src=["']([^"']+)["']`)
		submatch := srcRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		src := submatch[1]
		if strings.HasPrefix(src, "data:") || isRemote(src) {
			return match
		}

		target := filepath.Join(baseDir, src)

		resContent, err := os.ReadFile(target)
		if err != nil {
			log.Printf("Warning: Failed to fetch image resource %s: %v\n", target, err)
			return match
		}

		mimeType := http.DetectContentType(resContent)
		base64Content := base64.StdEncoding.EncodeToString(resContent)
		dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Content)

		return srcRegex.ReplaceAllString(match, fmt.Sprintf(`src="%s"`, dataURL))
	})

	return content
}

func run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("expect 2 args or higher")
	}

	outputFileName := args[0]
	inputFileNames := args[1:]

	// Determine the common root directory to create relative paths for the tree
	var commonRoot string

	absPath, err := filepath.Abs(inputFileNames[0])
	if err != nil {
		return fmt.Errorf("Failed to get absolute path: %v", err)
	}
	commonRoot = filepath.Dir(absPath)

	for _, name := range inputFileNames[1:] {
		absPath, err := filepath.Abs(name)
		if err != nil {
			return fmt.Errorf("Failed to get absolute path: %w", err)
		}
		dir := filepath.Dir(absPath)
		for !strings.HasPrefix(commonRoot, dir) && commonRoot != "." && commonRoot != "/" {
			commonRoot = filepath.Dir(commonRoot)
			if commonRoot == dir {
				break
			}
		}
	}

	var files []FileContent
	var targetFiles []string
	for _, name := range inputFileNames {
		info, err := os.Stat(name)
		if err != nil {
			return fmt.Errorf("Error stating %s: %v\n", name, err)
		}

		if info.IsDir() {
			err := filepath.WalkDir(name, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() && !strings.HasPrefix(d.Name(), ".") {
					absPath, err := filepath.Abs(path)
					if err != nil {
						return fmt.Errorf("cannot resolve absolute path: %w", err)
					}
					targetFiles = append(targetFiles, absPath)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("Error walking directory %s: %v\n", name, err)
			}
		} else {
			targetFiles = append(targetFiles, name)
		}
	}

	for _, inputFileName := range targetFiles {
		contentBytes, err := os.ReadFile(inputFileName)
		if err != nil {
			return fmt.Errorf("Error reading file %s: %w", inputFileName, err)
		}

		contentType := http.DetectContentType(contentBytes)
		var processedContent string

		// Determine how to process the file based on its type
		if strings.HasPrefix(contentType, "text/html") || strings.ToLower(filepath.Ext(inputFileName)) == ".html" {
			processedContent = inlineResources(inputFileName, string(contentBytes))
		} else if strings.HasPrefix(contentType, "image/") {
			base64Content := base64.StdEncoding.EncodeToString(contentBytes)
			processedContent = fmt.Sprintf(`<!DOCTYPE html><html><body style="margin:0;display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f0f0;"><img src="data:%s;base64,%s" style="max-width:100%%;max-height:100%%;object-fit:contain;"></body></html>`, contentType, base64Content)
		} else {
			// Treat as plain text and wrap in <pre>
			processedContent = fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head><body style="margin:0;padding:10px;"><pre style="white-space: pre-wrap; word-wrap: break-word; font-family: monospace;">%s</pre></body></html>`, html.EscapeString(string(contentBytes)))
		}

		// Escape for srcdoc
		relPath, err := filepath.Rel(commonRoot, inputFileName)
		if err != nil {
			return fmt.Errorf("could not find relative path for %s: %w", inputFileName, err)
		}

		files = append(files, FileContent{
			Name:       relPath,
			RawContent: processedContent,
		})
	}

	if len(files) == 0 {
		return fmt.Errorf("No valid input files processed.")
	}

	data := TemplateData{
		Files: files,
	}

	tmpl, err := template.New("main").Parse(mainTemplate)
	if err != nil {
		return fmt.Errorf("Error parsing template: %w", err)
	}

	var renderedHTML bytes.Buffer
	err = tmpl.Execute(&renderedHTML, data)
	if err != nil {
		return fmt.Errorf("Error executing template: %w", err)
	}

	err = os.WriteFile(outputFileName, renderedHTML.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Error writing output file %s: %w", outputFileName, err)
	}

	fmt.Printf("Successfully combined %d files into %s\n", len(files), outputFileName)
	return nil
}

func NewVersionCommand() *cobra.Command {
	var detail bool

	c := &cobra.Command{
		Use:   "version",
		Short: "show version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return version(cmd.OutOrStdout(), detail)
		},
	}
	c.Flags().BoolVarP(&detail, "detail", "d", false, "show details")

	return c
}

func version(w io.Writer, detail bool) error {
	path, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot get executable path: %w", err)
	}

	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read buildinfo: %w", err)
	}

	fmt.Fprintf(w, "go version: %s\n", info.GoVersion)
	fmt.Fprintf(w, "path: %s\n", info.Path)
	fmt.Fprintf(w, "mod: %s\n", info.Main.Path)
	fmt.Fprintf(w, "module version: %s\n", info.Main.Version)

	if detail {
		fmt.Fprintln(w)
		fmt.Fprintln(w, info)
	}

	return nil
}
