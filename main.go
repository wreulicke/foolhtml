package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// FileContent holds the name and escaped HTML content of a file.
type FileContent struct {
	Name        string
	EscapedHTML string
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
    <title>Combined HTML Files</title>
    <style>
        body { font-family: sans-serif; margin: 0; padding: 0; }
        .tabs {
            overflow: hidden;
            border-bottom: 1px solid #ccc;
            background-color: #f1f1f1;
        }
        .tabs button {
            background-color: inherit;
            float: left;
            border: none;
            outline: none;
            cursor: pointer;
            padding: 14px 16px;
            transition: 0.3s;
            font-size: 17px;
        }
        .tabs button:hover {
            background-color: #ddd;
        }
        .tabs button.active {
            background-color: #ccc;
        }
        .iframe-container {
            display: none;
            height: calc(100vh - 50px); /* Adjust based on tab height */
            width: 100%;
        }
        .iframe-container.active {
            display: block;
        }
        iframe {
            width: 100%;
            height: 100%;
            border: none;
        }
    </style>
</head>
<body>

    <div class="tabs">
        {{range $index, $file := .Files}}
            <button class="tablinks" onclick="openTab(event, 'tab-{{$index}}')">{{$file.Name}}</button>
        {{end}}
    </div>

    {{range $index, $file := .Files}}
        <div id="tab-{{$index}}" class="iframe-container">
            <iframe srcdoc="{{$file.EscapedHTML}}" sandbox="allow-scripts allow-same-origin"></iframe>
        </div>
    {{end}}

    <script>
        function openTab(evt, tabId) {
            var i, iframeContainers, tablinks;

            iframeContainers = document.getElementsByClassName("iframe-container");
            for (i = 0; i < iframeContainers.length; i++) {
                iframeContainers[i].style.display = "none";
                iframeContainers[i].classList.remove("active");
            }

            tablinks = document.getElementsByClassName("tablinks");
            for (i = 0; i < tablinks.length; i++) {
                tablinks[i].classList.remove("active");
            }

            document.getElementById(tabId).style.display = "block";
            document.getElementById(tabId).classList.add("active");
            evt.currentTarget.classList.add("active");
        }

        // Open the first tab by default
        document.addEventListener("DOMContentLoaded", function() {
            document.querySelector(".tablinks").click();
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

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func isRemote(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "//")
}

func fetchResource(target string) ([]byte, error) {
	if strings.HasPrefix(target, "//") {
		target = "https:" + target
	}
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		resp, err := httpClient.Get(target)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad status: %s", resp.Status)
		}
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(target)
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
		target := href
		if !isRemote(href) {
			target = filepath.Join(baseDir, href)
		}

		resContent, err := fetchResource(target)
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
		target := src
		if !isRemote(src) {
			target = filepath.Join(baseDir, src)
		}

		resContent, err := fetchResource(target)
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
		if strings.HasPrefix(src, "data:") {
			return match
		}

		target := src
		if !isRemote(src) {
			target = filepath.Join(baseDir, src)
		}

		resContent, err := fetchResource(target)
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
		fmt.Printf("Usage: %s <output_file.html> <input_file1.html> [input_file2.html...]\n", os.Args[0])
		os.Exit(1)
	}

	outputFileName := os.Args[1]
	inputFileNames := os.Args[2:]

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
					targetFiles = append(targetFiles, path)
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
			processedContent = fmt.Sprintf("<!DOCTYPE html><html><body style=\"margin:0;display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f0f0;\"><img src=\"data:%s;base64,%s\" style=\"max-width:100%%;max-height:100%%;object-fit:contain;\"></body></html>", contentType, base64Content)
		} else {
			// Treat as plain text and wrap in <pre>
			processedContent = fmt.Sprintf("<!DOCTYPE html><html><head><meta charset=\"UTF-8\"></head><body style=\"margin:0;padding:10px;\"><pre style=\"white-space: pre-wrap; word-wrap: break-word; font-family: monospace;\">%s</pre></body></html>", html.EscapeString(string(contentBytes)))
		}

		// Escape for srcdoc
		escapedContent := html.EscapeString(processedContent)

		files = append(files, FileContent{
			Name:        filepath.Base(inputFileName),
			EscapedHTML: escapedContent,
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
