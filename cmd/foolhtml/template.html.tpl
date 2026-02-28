<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Combined Files</title>
    <style>
        *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            display: flex;
            height: 100vh;
            overflow: hidden;
        }
        #sidebar {
            width: 240px;
            min-width: 160px;
            background: #252526;
            border-right: 1px solid #1a1a1a;
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }
        #sidebar-header {
            padding: 9px 14px;
            font-size: 11px;
            font-weight: 700;
            letter-spacing: 0.1em;
            text-transform: uppercase;
            color: #aaa;
            border-bottom: 1px solid #3a3a3a;
            flex-shrink: 0;
        }
        #file-tree {
            flex-grow: 1;
            overflow-y: auto;
            padding: 4px 0;
            font-size: 13px;
            color: #ccc;
        }
        #file-tree::-webkit-scrollbar { width: 6px; }
        #file-tree::-webkit-scrollbar-track { background: transparent; }
        #file-tree::-webkit-scrollbar-thumb { background: #555; border-radius: 3px; }
        .tree-row {
            display: flex;
            align-items: center;
            height: 22px;
            cursor: pointer;
            white-space: nowrap;
            overflow: hidden;
            border-radius: 3px;
            margin: 1px 6px;
            color: #ccc;
        }
        .tree-row:hover { background: #2a2d2e; }
        .tree-row.active { background: #094771; color: #fff; }
        .tree-row .chevron {
            width: 16px;
            flex-shrink: 0;
            color: #858585;
            font-size: 10px;
            text-align: center;
            transition: transform 0.12s ease;
            display: inline-block;
        }
        .tree-row.open > .chevron { transform: rotate(90deg); }
        .tree-row .icon {
            width: 18px;
            flex-shrink: 0;
            font-size: 13px;
            text-align: center;
            margin-right: 5px;
        }
        .tree-row .label {
            overflow: hidden;
            text-overflow: ellipsis;
            flex-grow: 1;
        }
        .tree-children.collapsed { display: none; }
        #content-viewer { flex-grow: 1; }
        iframe { width: 100%; height: 100%; border: none; display: block; }
        #download {
            position: fixed;
            right: 18px;
            top: 18px;
            width: 50px;
            height: 50px;
            font-size: 1.5rem;
            background: hsl(189, 73%, 64%);
            border: 0px;
            border-radius: 9999px;
            cursor: pointer;
        }
        #download:hover {
            background: hsl(189, 73%, 64%, 0.7);
            border: 1px solid;
            border-color: hsl(189, 7%, 26%, 0.3);
        }
    </style>
</head>
<body>
    <div id="sidebar">
        <div id="sidebar-header">Explorer</div>
        <div id="file-tree"></div>
    </div>
    <button id="download">⬇️</button>
    <div id="content-viewer">
        <iframe id="content-frame" sandbox="allow-scripts allow-same-origin"></iframe>
    </div>
    <script>
        const files = [
            {{range $i, $f := .Files}}
            { path: "{{$f.Path}}", name: "{{$f.Name}}", base64: "{{ $f.Base64 }}", html: "{{ $f.PreviewContent }}" },
            {{end}}
        ];
        console.log(files)
        try {
            console.log(atob(files[0].base64))
        } catch (e) {}

        const EXT_ICONS = {
            html: ['\u{1F310}', '#e44d26'], htm:  ['\u{1F310}', '#e44d26'],
            css:  ['\u{1F3A8}', '#264de4'],
            js:   ['\u26A1', '#f0db4f'], ts:   ['\u26A1', '#3178c6'],
            json: ['{}', '#cbcb41'],
            md:   ['\u270F', '#519aba'], markdown: ['\u270F', '#519aba'],
            png:  ['\u{1F5BC}', '#a8cc8c'], jpg:  ['\u{1F5BC}', '#a8cc8c'],
            jpeg: ['\u{1F5BC}', '#a8cc8c'], gif:  ['\u{1F5BC}', '#a8cc8c'],
            svg:  ['\u{1F5BC}', '#a8cc8c'], webp: ['\u{1F5BC}', '#a8cc8c'],
            pdf:  ['\u{1F4C4}', '#f15642'],
        };

        function fileIcon(name) {
            const ext = name.includes('.') ? name.split('.').pop().toLowerCase() : '';
            return EXT_ICONS[ext] || ['\u{1F4C4}', '#89c4e1'];
        }

        function buildTree(files) {
            const root = { dirs: {}, files: [] };
            for (const f of files) {
                const parts = f.path.replace(/\\/g, '/').split('/');
                let node = root;
                for (let i = 0; i < parts.length - 1; i++) {
                    if (!node.dirs[parts[i]]) node.dirs[parts[i]] = { dirs: {}, files: [] };
                    node = node.dirs[parts[i]];
                }
                node.files.push({ name: parts[parts.length - 1], file: f });
            }
            return root;
        }

        let activeRow = null;
        const frame = document.getElementById('content-frame');

        let openFile = null;
        function renderNode(node, depth) {
            const frag = document.createDocumentFragment();

            for (const name of Object.keys(node.dirs).sort()) {
                const row = document.createElement('div');
                row.className = 'tree-row open';
                row.style.paddingLeft = (depth * 16) + 'px';

                const chevron = document.createElement('span');
                chevron.className = 'chevron';
                chevron.textContent = '\u25B6';

                const icon = document.createElement('span');
                icon.className = 'icon';
                icon.textContent = '\u{1F4C2}';

                const label = document.createElement('span');
                label.className = 'label';
                label.textContent = name;

                row.appendChild(chevron);
                row.appendChild(icon);
                row.appendChild(label);

                const childrenDiv = document.createElement('div');
                childrenDiv.className = 'tree-children';
                childrenDiv.appendChild(renderNode(node.dirs[name], depth + 1));

                row.addEventListener('click', () => {
                    const isOpen = row.classList.toggle('open');
                    childrenDiv.classList.toggle('collapsed', !isOpen);
                    icon.textContent = isOpen ? '\u{1F4C2}' : '\u{1F4C1}';
                });

                frag.appendChild(row);
                frag.appendChild(childrenDiv);
            }

            for (const { name, file } of node.files.slice().sort((a, b) => a.name.localeCompare(b.name))) {
                const row = document.createElement('div');
                row.className = 'tree-row';
                row.style.paddingLeft = (depth * 16 + 16) + 'px';

                const [iconChar, color] = fileIcon(name);
                const icon = document.createElement('span');
                icon.className = 'icon';
                icon.textContent = iconChar;
                icon.style.color = color;

                const label = document.createElement('span');
                label.className = 'label';
                label.textContent = name;

                row.appendChild(icon);
                row.appendChild(label);

                row.addEventListener('click', () => {
                    if (activeRow) activeRow.classList.remove('active');
                    row.classList.add('active');
                    activeRow = row;
                    openFile = file;
                    frame.srcdoc = file.html;
                });

                frag.appendChild(row);
            }

            return frag;
        }
        const dEl = document.getElementById('download')
        dEl.addEventListener('click', (e) => {
            const d = atob(openFile.base64);
            const array = new Uint8Array(Array.prototype.map.call(d, c => c.charCodeAt()))
            const blob = new Blob([array], {type: "application/octet-stream"})
            var url = window.URL.createObjectURL(blob);
            var a = document.createElement("a");
            a.href = url;
            a.download = openFile.name;
            a.click();
        })

        const treeEl = document.getElementById('file-tree');
        treeEl.appendChild(renderNode(buildTree(files), 0));

        // Select first file by default
        const first = treeEl.querySelector('.tree-row:not(.open)');
        if (first) first.click();
    </script>
</body>
</html>