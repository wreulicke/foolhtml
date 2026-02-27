# foolhtml

`foolhtml` is a simple Go utility that combines multiple files (HTML, images, text, etc.) into a single, portable HTML file. It uses a tabbed interface to switch between the included files and automatically inlines local resources (CSS, JavaScript, and images) for HTML content, while preserving remote links.

## Features

- **Recursive Processing:** If a directory is provided as an input, the tool recursively finds all files (excluding hidden ones).
- **Multi-format Support:**
  - **HTML:** Inlines resources and renders as usual.
  - **Images:** Automatically converts to Base64 and displays them.
  - **Text/Code:** Wraps non-HTML files (like JS, CSS, or logs) in `<pre>` tags for easy viewing.
- **Tabbed Interface:** Combines everything into one file with clean tab-based navigation.
- **Resource Inlining:** For HTML files, it automatically fetches and inlines **local** resources:
  - Local CSS via `<link rel="stylesheet">`
  - Local JavaScript via `<script src="...">`
  - Local Images via `<img src="...">` (converted to Base64 data URLs)
  - *Note: Remote resources (starting with `http://`, `https://`, or `//`) are not inlined and remain as external links.*
- **Isolated Execution:** Uses `<iframe>` with the `sandbox` attribute to display content, providing a layer of isolation.

## Installation

Ensure you have Go installed on your system, then build the binary:

```bash
go build -o foolhtml main.go
```

## Usage

```bash
./foolhtml <output_file.html> <input_path1> [input_path2...]
```

### Example

```bash
./foolhtml examples/output.html test_files/
```

This will create `examples/output.html` containing all files found in `test_files/`, with each accessible via a tab.

## Security Considerations

**Important:** This tool is intended for merging trusted files. Be aware of the following:

- **Path Traversal:** The tool currently does not strictly validate paths for local resources, which could lead to arbitrary file inclusion if processing untrusted input.
- **Resource Limits:** There are no limits on the size of resources fetched, which could lead to high memory usage (DoS) with malicious inputs.
- **Iframe Sandbox:** Content is rendered in an iframe with `sandbox="allow-scripts allow-same-origin"`. While this provides some isolation, scripts in the input files will still execute.

## License

MIT
