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

Download the latest binary from the [GitHub Releases](https://github.com/wreulicke/foolhtml/releases/latest) page.

### macOS

```bash
# Apple Silicon (arm64)
VERSION=$(curl -s https://api.github.com/repos/wreulicke/foolhtml/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L "https://github.com/wreulicke/foolhtml/releases/download/${VERSION}/foolhtml_${VERSION#v}_darwin_arm64" -o foolhtml
chmod +x foolhtml && mv foolhtml /usr/local/bin/foolhtml

# Intel (amd64)
VERSION=$(curl -s https://api.github.com/repos/wreulicke/foolhtml/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L "https://github.com/wreulicke/foolhtml/releases/download/${VERSION}/foolhtml_${VERSION#v}_darwin_amd64" -o foolhtml
chmod +x foolhtml && mv foolhtml /usr/local/bin/foolhtml
```

### Linux

```bash
# amd64
VERSION=$(curl -s https://api.github.com/repos/wreulicke/foolhtml/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L "https://github.com/wreulicke/foolhtml/releases/download/${VERSION}/foolhtml_${VERSION#v}_linux_amd64" -o foolhtml
chmod +x foolhtml && mv foolhtml /usr/local/bin/foolhtml

# arm64
VERSION=$(curl -s https://api.github.com/repos/wreulicke/foolhtml/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L "https://github.com/wreulicke/foolhtml/releases/download/${VERSION}/foolhtml_${VERSION#v}_linux_arm64" -o foolhtml
chmod +x foolhtml && mv foolhtml /usr/local/bin/foolhtml
```

### Windows

```bash
# amd64
VERSION=$(curl -s https://api.github.com/repos/wreulicke/foolhtml/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L "https://github.com/wreulicke/foolhtml/releases/download/${VERSION}/foolhtml_${VERSION#v}_windows_amd64.exe" -o foolhtml.exe

# arm64
VERSION=$(curl -s https://api.github.com/repos/wreulicke/foolhtml/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L "https://github.com/wreulicke/foolhtml/releases/download/${VERSION}/foolhtml_${VERSION#v}_windows_arm64.exe" -o foolhtml.exe
```

### Docker

```bash
docker run --rm -v $(pwd):/work ghcr.io/wreulicke/foolhtml output.html input/
```

### GitHub Actions

Use the `wreulicke/foolhtml` action in your workflow. The following example generates a combined HTML report and uploads it as a workflow artifact. On pull requests, a comment with the artifact URL is posted automatically.
You can see artifacts by just clicking the link.

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - uses: wreulicke/foolhtml@main
        with:
          output: /tmp/output.html
          inputs: |
            path/to/file1.html
            path/to/dir/

      - uses: actions/upload-artifact@v7
        id: upload-artifact
        with:
          name: report
          archive: false
          path: /tmp/output.html
      
      - uses: actions/github-script@v6
        if: github.event_name == 'pull_request'
        env:
          ARTIFACT_URL: ${{ steps.upload-artifact.outputs.artifact-url }}
        with:
          script: |
            const artifactUrl = process.env.ARTIFACT_URL;
            const commentBody = `You can see [ci-reports](${artifactUrl}) here.`;
            await github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: commentBody,
            });
```

Available inputs:

| Input          | Description                                      | Required | Default               |
|----------------|--------------------------------------------------|----------|-----------------------|
| `output`       | Output HTML file path                            | Yes      |                       |
| `inputs`       | Newline-separated list of input files/directories | Yes      |                       |
| `version`      | foolhtml version to use (e.g. `v0.0.6`)          | No       | `latest`              |
| `github-token` | GitHub token to avoid API rate limiting          | No       | `${{ github.token }}` |

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
