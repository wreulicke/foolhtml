package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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

const mainTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Combined Files</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            margin: 0;
            display: flex;
            height: 100vh;
            overflow: hidden;
        }
        #file-tree {
            width: 250px;
            border-right: 1px solid #ccc;
            padding: 10px;
            overflow-y: auto;
            background-color: #f7f7f7;
        }
        #file-tree ul {
            list-style: none;
            padding-left: 15px;
            margin: 0;
        }
        #file-tree li {
            padding: 4px 0;
        }
        #file-tree a {
            text-decoration: none;
            color: #333;
            cursor: pointer;
            display: block;
            padding: 2px 5px;
            border-radius: 3px;
        }
        #file-tree a:hover {
            background-color: #e0e0e0;
        }
        #file-tree a.active {
            background-color: #007bff;
            color: white;
        }
        #content-viewer {
            flex-grow: 1;
            display: flex;
        }
        iframe {
            width: 100%;
            height: 100%;
            border: none;
        }
    </style>
</head>
<body>
    <div id="file-tree">
        <ul>
            {{range $index, $file := .Files}}
                <li><a href="#" data-index="{{$index}}">{{$file.Name}}</a></li>
            {{end}}
        </ul>
    </div>
    <div id="content-viewer">
        <iframe id="content-frame" sandbox="allow-scripts allow-same-origin"></iframe>
    </div>

    <script>
		const files = [
			{{range .Files}}
			{
				escapedHtml: "{{.RawContent}}",
			},
			{{end}}
		];

        document.addEventListener("DOMContentLoaded", function() {
            const fileLinks = document.querySelectorAll("#file-tree a");
            const contentFrame = document.getElementById("content-frame");
			let activeLink = null;

            fileLinks.forEach(link => {
                link.addEventListener("click", function(e) {
                    e.preventDefault();

					if (activeLink) {
						activeLink.classList.remove("active");
					}
					this.classList.add("active");
					activeLink = this;

                    const index = parseInt(this.getAttribute("data-index"), 10);
					if (index >= 0 && index < files.length) {
						contentFrame.srcdoc = files[index].escapedHtml;
					}
                });
            });

            // Open the first file by default
            if (fileLinks.length > 0) {
                fileLinks[0].click();
            }
        });
    </script>
</body>
</html>`

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

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <output_file.html> <input_path1> [input_path2...]\n", os.Args[0])
		os.Exit(1)
	}

	outputFileName := os.Args[1]
	inputFileNames := os.Args[2:]

	// Determine the common root directory to create relative paths for the tree
	var commonRoot string

	absPath, err := filepath.Abs(inputFileNames[0])
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}
	commonRoot = filepath.Dir(absPath)

	for _, name := range inputFileNames[1:] {
		absPath, err := filepath.Abs(name)
		if err != nil {
			log.Fatalf("Failed to get absolute path: %v", err)
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
			log.Printf("Error stating %s: %v\n", name, err)
			continue
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
				log.Printf("Error walking directory %s: %v\n", name, err)
			}
		} else {
			targetFiles = append(targetFiles, name)
		}
	}

	for _, inputFileName := range targetFiles {
		contentBytes, err := os.ReadFile(inputFileName)
		if err != nil {
			log.Printf("Error reading file %s: %v\n", inputFileName, err)
			continue
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
			log.Printf("Warning: could not find relative path for %s: %v", inputFileName, err)
			relPath = filepath.Base(inputFileName)
		}

		files = append(files, FileContent{
			Name:       relPath,
			RawContent: processedContent,
		})
	}

	if len(files) == 0 {
		log.Fatal("No valid input files processed.")
	}

	data := TemplateData{
		Files: files,
	}

	tmpl, err := template.New("main").Parse(mainTemplate)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	var renderedHTML bytes.Buffer
	err = tmpl.Execute(&renderedHTML, data)
	if err != nil {
		log.Fatalf("Error executing template: %v", err)
	}

	err = os.WriteFile(outputFileName, renderedHTML.Bytes(), 0644)
	if err != nil {
		log.Fatalf("Error writing output file %s: %v", outputFileName, err)
	}

	fmt.Printf("Successfully combined %d files into %s\n", len(files), outputFileName)
}
