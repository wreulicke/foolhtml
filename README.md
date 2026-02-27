# foolhtml

`foolhtml` is a simple Go utility that combines multiple HTML files into a single, portable HTML file. It uses a tabbed interface to switch between the included files and automatically inlines external resources (CSS, JavaScript, and images) to ensure the output is self-contained.

## Features

- **Recursive Processing:** If a directory is provided as an input, the tool recursively finds all files (excluding hidden ones).
- **Multi-format Support:**
  - **HTML:** Inlines resources and renders as usual.
  - **Images:** Automatically converts to Base64 and displays them.
  - **Text/Code:** Wraps non-HTML files (like JS, CSS, or logs) in `<pre>` tags for easy viewing.
- **Tabbed Interface:** Combines everything into one file with clean tab-based navigation.
- **Resource Inlining:** Automatically fetches and inlines:
  - External CSS via `<link rel="stylesheet">`
  - External JavaScript via `<script src="...">`
  - Images via `<img src="...">` (converted to Base64 data URLs)
- **Isolated Execution:** Uses `<iframe>` with the `sandbox` attribute to display content, providing a layer of isolation.

## Installation

Ensure you have Go installed on your system, then build the binary:

```bash
go build -o foolhtml main.go
```

## Usage

```bash
./foolhtml <output_file.html> <input_file1.html> [input_file2.html...]
```

### Example

```bash
./foolhtml examples/output.html test_files/file1.html test_files/file2.html
```

This will create `examples/output.html` containing the content of the two input files, with each accessible via a tab.

## Security Considerations

**Important:** This tool is intended for merging trusted HTML files. Be aware of the following:

- **Path Traversal:** The tool currently does not strictly validate paths for local resources, which could lead to arbitrary file inclusion if processing untrusted input.
- **SSRF (Server-Side Request Forgery):** It fetches remote resources (http/https) defined in the input HTML. Running this on untrusted files could allow an attacker to make requests to internal services.
- **Resource Limits:** There are no limits on the size of resources fetched, which could lead to high memory usage (DoS) with malicious inputs.
- **Iframe Sandbox:** Content is rendered in an iframe with `sandbox="allow-scripts allow-same-origin"`. While this provides some isolation, scripts in the input files will still execute.

## License

MIT
