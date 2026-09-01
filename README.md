<div align="center">

# 🪨 gobsidian

**MCP server for local Obsidian vaults. One Go binary, no runtime, no orphan processes.**

🌍 **English** · [Português](README.pt-BR.md)

[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2025--11--25-6E56CF)](https://modelcontextprotocol.io)
[![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20Linux%20%7C%20macOS-informational)](#-compatibility)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

🔍 [Overview](#-overview) ·
📦 [Install](#-install) ·
⚙️ [Configuration](#-configuration) ·
🧰 [Tools](#-mcp-tools) ·
💻 [CLI](#-command-line)

🧵 [Daemon](#-daemon) ·
🖥️ [Compatibility](#-compatibility) ·
📊 [Performance](#-performance) ·
🛠️ [Development](#-development) ·
📚 [Documentation](#-documentation)

</div>

---

## 🔍 Overview

`gobsidian` exposes a local Obsidian vault to any MCP host — Claude Desktop, Claude Code, Gemini CLI, Antigravity, VS Code — through a single executable speaking JSON-RPC over stdio.

It came out of three concrete problems with the existing Obsidian MCP servers:

| 🐛 Problem | ✅ What `gobsidian` does |
|---|---|
| Zombie processes after the host closes | Four independent shutdown mechanisms, each verified over 100 abrupt-kill cycles |
| Full reindex on every file event | Incremental watcher with coalescing, and an index of byte offsets |
| Generic parsers that break on wikilinks, embeds and block ids | Purpose-built parser, frozen by 48 golden files and checked against a real dump of Obsidian's `metadataCache` |

### ✨ Features

- ⚡ **Reading by offset.** Reading a 2 KB section out of a 500 KB note costs 2 KB of I/O, not 500 KB.
- 🎯 **Search tells you WHERE.** `vault_search` returns `match_offset`, the absolute offset of the match, which feeds `note_read(offset=…)` directly — find a term in a 255 KB note and read only the bytes around it.
- 🗺️ **Converted notes get their structure back.** PDF, DOCX and EPUB become notes with no `#` heading at all: the title is a bold paragraph. `note_outline` maps them, separating real headings from **candidates** and saying which is which, and `note_read(heading=…)` reads a candidate's section directly when no Markdown heading matches — the reply carries `section_synthetic` so a guess is never passed off as structure. Writing stays out on purpose: reading the wrong place returns the wrong paragraph, writing the wrong place destroys work.
- 📚 **Batch reads with per-item overrides.** `note_read` accepts `["a.md", {"path":"b.md","heading":"X"}]` in one list: six chapters with six sections in one call, not six.
- 🔎 **BM25 search** with field weights, exact phrase, and folder, tag, frontmatter and date filters.
- 🇧🇷 **Portuguese analyzer**: accents, case folding, and dual indexing — raw and reduced forms in the same posting list. Index keys normalize to NFC, so a note written on a Mac is found by a request from Windows.
- ✍️ **Surgical, atomic writes.** `note_append` and `note_patch` insert by heading or block id; every write goes through a temp file and a rename, so an Obsidian open beside it never sees a half-written file.
- 🔗 **`note_move` rewrites the links**, preserving alias, anchor and original form.
- 👀 **Incremental watcher** with debounce, and reconciliation by full scan when `fsnotify` overflows.
- 🔒 **`--read-only` mode** that removes the write tools from `ListTools` — absent, not merely rejected.
- 🚫 **No socket that leaves the machine**, enforced in CI by a `go vet` analyzer on all three systems.

> [!NOTE]
> The vault is the source of truth; the index is derived and disposable. If it corrupts, it rebuilds in seconds.

---

## 📦 Install

Either installer downloads the release binary, **verifies its SHA-256 and aborts on a mismatch**, installs it, adds it to `PATH`, reads Obsidian's own vault registry, and asks which hosts to register the server with.

| You are on | Use | Requires |
|---|---|---|
| 🪟 Windows | `install.ps1` at the root | nothing beyond PowerShell |
| 🪟 🐧 🍎 Windows, Linux or macOS | the `installer/` folder | Node.js 18+ |

**Windows, no dependencies:**

```powershell
iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/install.ps1)
```

**Cross-platform** — the same installer in Node, with no `npm install` and no `node_modules`: <!-- check-doc-refs: ignore node_modules -- diretorio do Node, citado para dizer que o instalador nao cria um -->

```powershell
iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.ps1)
```

```bash
curl -fsSL https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.bash | bash
```

<details>
<summary>⚙️ <b>Options</b> — unattended install, specific host, pinned version</summary>

<br>

`iex` runs the **text** of the script, and a `param()` block receives nothing that way. To pass options, use a scriptblock:

```powershell
& ([scriptblock]::Create((irm .../install.ps1))) -Vault "C:\My Vault" -Hosts claude-desktop -ReadOnly
& ([scriptblock]::Create((irm .../installer/install.ps1))) --vault "C:\My Vault" --hosts claude-desktop
```

```bash
curl -fsSL .../installer/install.bash | bash -s -- --vault "/path/to/vault" --yes
```

| Root | `installer/` | Effect |
|---|---|---|
| `-Vault` | `--vault` | Vault to serve. Repeatable, or several comma-separated. |
| `-Hosts` | `--hosts` | Configure only these hosts, no menu. |
| `-Version` | `--version` | A specific release. Default: the latest. |
| `-InstallDir` | `--install-dir` | Where to put the binary. |
| `-ReadOnly` | `--read-only` | Register with `--read-only`. |
| `-Yes` | `--yes`, `-y` | Ask nothing. Requires the vault. |
| `-NoPath` | `--no-path` | Leave `PATH` alone. |
| — | `--force` | Reinstall even if already present. |

Accepted hosts: `claude-desktop`, `claude-code`, `gemini-cli`, `antigravity`, `antigravity-ide`, `codex`, `vscode`, `cursor`, `windsurf`.

Environment variables work in both installers too: `GOBSIDIAN_VAULT`, `GOBSIDIAN_VERSION`, `GOBSIDIAN_INSTALL_DIR`.

</details>

<details>
<summary>📥 <b>Prebuilt binary or from source</b></summary>

<br>

Download from [Releases](https://github.com/jonyduque/Gobsidian/releases) and put it in any directory on `PATH`. There is no service and no system registration.

| System | File |
|---|---|
| 🪟 Windows x86-64 | `gobsidian-windows-amd64.exe` |
| 🐧 Linux x86-64 | `gobsidian-linux-amd64` |
| 🍎 macOS Apple Silicon | `gobsidian-darwin-arm64` |

```bash
sha256sum -c SHA256SUMS.txt --ignore-missing
```

From source, with **Go 1.25+** (a floor imposed by the MCP SDK, not a preference):

```bash
go install github.com/jonyd/gobsidian/cmd/gobsidian@latest
```

</details>

> [!TIP]
> With elevation it goes to `Program Files` and the machine `PATH`; without elevation, to `%LOCALAPPDATA%\Programs\gobsidian` and the user `PATH`. The installer does **not** request UAC on its own.

> [!NOTE]
> **Upgrading from an older version?** The cache format has changed, so the first start rebuilds the index in the background — the tools answer from the first second. Do not keep two versions installed side by side: each invalidates the other's cache on every switch.

---

## ⚙️ Configuration

The installer does this for you. What follows is for configuring by hand.

<details>
<summary>📝 <b>Manual configuration per host</b></summary>

<br>

**Claude Desktop** — `%APPDATA%\Claude\claude_desktop_config.json` on Windows, `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS:

```json
{
  "mcpServers": {
    "gobsidian": {
      "command": "C:\\Program Files\\gobsidian\\gobsidian.exe",
      "args": ["serve", "--vault", "C:\\My Vault"]
    }
  }
}
```

**Claude Code** and **Gemini CLI** register themselves — the argument order differs:

```bash
claude mcp add gobsidian --scope user -- gobsidian serve --vault "C:\My Vault"
gemini mcp add gobsidian gobsidian serve --vault "C:\My Vault" --scope user
```

**VS Code**:

```bash
code --add-mcp '{"name":"gobsidian","command":"gobsidian","args":["serve","--vault","C:\\My Vault"]}'
```

**Antigravity**, **Antigravity IDE**, **Cursor** and **Windsurf** use the same format as Claude Desktop, in `~/.gemini/antigravity/mcp_config.json`, `~/.gemini/antigravity-ide/mcp_config.json`, `~/.cursor/mcp.json` and `~/.codeium/windsurf/mcp_config.json`.

> [!IMPORTANT]
> Three mistakes account for most Windows failures: **single backslashes** (JSON requires `\\`), **extra quotes** around a path with spaces (each `args` entry is already one string), and a **relative path** to the binary (the host does not inherit your shell's `PATH`).

</details>

<details>
<summary>🎛️ <b><code>serve</code> options</b></summary>

<br>

| Flag | Effect |
|---|---|
| `--vault <path>` | Vault root. Required. |
| `--read-only` | Removes the entire write surface. |
| `--cache-dir <path>` | Cache directory. Default: a hash of the vault path, always **outside** it. |
| `--debounce-ms <n>` | Watcher coalescing window. |
| `--log-level <level>` | `debug`, `info`, `warn` or `error`. |
| `--eager-search` | Loads the search index at boot. Default: lazy — most sessions read and write without ever searching. |

</details>

> [!TIP]
> Start with `--read-only` until you trust the configuration. You keep search, reading and the link graph, with no tool able to touch the disk.

---

## 🧰 MCP tools

Full contracts, schemas and error codes in [`docs/TOOLS.md`](docs/TOOLS.md).

**📖 Reading**

| Tool | What it does |
|---|---|
| `vault_search` | BM25 search, exact phrase, folder, tag, frontmatter and date filters |
| `note_read` | A whole note, a section by heading, or a block by `^id`; several notes in one call, with per-item overrides |
| `note_outline` | The note's map: real headings, and title candidates for notes converted from PDF/DOCX/EPUB |
| `note_list` | Lists by glob, folder, tag or frontmatter query |
| `note_metadata` | Frontmatter, tags, links, backlinks, headings and blocks |
| `link_graph` | Link neighbourhood, with direction and depth |
| `tag_list` | Every tag in the vault, with counts |
| `vault_stats` | Notes, orphans, broken links and watcher counters |

**✏️ Writing** — all accept `dry_run` and `expected_hash`

| Tool | What it does |
|---|---|
| `note_create` | Creates the note, failing if it already exists |
| `note_append` | Appends to the end of the note or of a section |
| `note_patch` | Replaces the content under a heading or block |
| `note_move` | Moves or renames, rewriting the wikilinks that point at it |
| `note_delete` | Removes, with a prior report of the links that will break |

Notes are also published as MCP resources under `gobsidian:///<path>`. The scheme is our own because `obsidian://` belongs to the application and is registered with the operating system.

---

## 💻 Command line

`gobsidian` works outside MCP, and that is what makes diagnosis possible.

```bash
gobsidian serve   --vault "/path/to/vault"                    # what the MCP host runs
gobsidian doctor  --vault "/path/to/vault"                    # diagnoses the environment
gobsidian index   --vault "/path/to/vault" --json             # indexes and exits, with a summary
gobsidian search  "architecture" --vault "/path/to/vault"     # search without an MCP host
gobsidian inspect "Folder/Note.md" --vault "/vault" --json    # how the note was interpreted
```

> [!TIP]
> 🩺 `gobsidian doctor` is the first command to run when something does not work: unreachable vault, permissions, cloud-only OneDrive files, paths over 260 characters, and casing collisions.

**Output.** Coloured in a terminal, plain text when redirected — `doctor > report.txt` writes a clean file. The decision is per destination, so a redirected `stdout` does not strip colour from `stderr`. `NO_COLOR` turns everything off. The markers `[OK]`, `[!]`, `[i]`, `[*]` and `[...]` are pure ASCII, and colour only reinforces them.

In `serve`, **stdout belongs entirely to JSON-RPC** and every log goes to stderr. `doctor` and `version` print to stdout on purpose, because they are CLI commands.

---

## 🧵 Daemon

On by default. When two or more sessions point at the **same vault at the same time** and one of them searches, the daemon shares a single index between them instead of each loading its own copy.

Aggregate working set, a real vault of 4,513 notes:

| Sessions | Without daemon | With daemon | |
|---|---|---|---|
| 1 | 244.3 MB | 260.3 MB | +6.5% |
| 3 | 733.3 MB | 288.7 MB | −60.6% |
| 5 | 1,221.3 MB | 319.4 MB | −73.8% |

A single isolated session — the most common case — **costs 16 MB more**. The measurement's technical recommendation was to ship it off by default; the project decision was to ship it on, so that the gain at three and five sessions does not depend on finding an environment variable. Full table and reasoning in [`docs/OPERACAO.md`](docs/OPERACAO.md).

```bash
GOBSIDIAN_NO_DAEMON=1 gobsidian serve --vault "/path/to/vault"
```

In `claude_desktop_config.json`, the variable goes in the server entry's `env` block.

The transport is a local Unix domain socket under the user's runtime directory, and ownership of the startup lock is a kernel lock — `flock` on Unix, `LockFileEx` on Windows — so a daemon that dies never leaves the lock held. **If the daemon fails to start, `serve` falls back to the usual in-process mode** — the worst case is losing the shared memory, never the functionality. With no session connected, the daemon exits on its own after 15 minutes.

---

## 🖥️ Compatibility

| Item | State |
|---|---|
| 🪟 Windows 10+ | First-class platform |
| 🐧 Linux (kernel 5.x+) and 🍎 macOS 13+ | Supported; CI runs build, `vet` and `go test -race` on all three |
| MCP protocol | `2025-11-25`; negotiation down to earlier versions by the official SDK, pinned at `v1.5.0` |
| ☁️ OneDrive, Dropbox and Google Drive | Supported, including cloud-only files, which are never opened so as not to trigger a download |
| Paths over 260 characters on Windows | Supported |

---

## 📊 Performance

Deterministic synthetic vault of **5,000 notes**, 12-core laptop running Windows 11, without `-race`.

| What | Measured |
|---|---|
| Cold indexing (metadata) | **500 ms** |
| Boot with a valid cache | 208–282 ms synthetic; 179–193 ms at 1,254 notes; 891 ms at 5,686 |
| `note_read` p95 | **345 µs** |
| `note_list` with a filter, p95 | **534 µs** |
| `vault_search` p95 | **8 of 8 shapes**, worst case 43 ms |
| Single-file reindex | **335 µs** |
| Live heap at rest | 5 real vaults, 15% to 64% of headroom |
| Orphan processes after the host dies | **0 over 400 cycles** |

Boot scales with the size of the vault: the `Stat` sweep that validates the cache before accepting it costs more in a cloud-synced vault. Memory is measured as **live heap**, not RSS — RSS in Go tracks the collector's goal, roughly 2× the live heap at the last cycle, and measures GC policy rather than the data held.

The table of all 22 non-functional requirements, each with a number or the words "not measured", is in [`docs/OPERACAO.md`](docs/OPERACAO.md).

CI compares six benchmarks against `docs/bench-baseline.json` and **fails the build** on a regression above 20%.

---

## 🛠️ Development

```bash
pwsh -File scripts/verify.ps1        # the whole battery, stopping at the first error
pwsh -File scripts/build.ps1         # binary with the version baked in
```

<details>
<summary>🔬 <b>What the battery covers</b></summary>

<br>

In order, stopping at the first error: build, `go test -race`, latency ceilings **without** `-race`, `go vet` on all three targets, `gofmt`, `golangci-lint`, the network check, the schema-parameter check, the check for documentation references to artifacts that do not exist, and the README anchor check.

It is one command because a loose list invites running three of the five.

> [!IMPORTANT]
> `golangci-lint` must be **v2.12.2**. A binary built with a Go older than `go.mod` rejects the config before analysing a single line, and a local zero says nothing about CI.

</details>

<details>
<summary>👻 <b>The orphan-process gate</b> — four scenarios, one per mechanism</summary>

<br>

```bash
pwsh -File scripts/test_orphans.ps1 -Cycles 100                      # all four, the default
pwsh -File scripts/test_orphans.ps1 -Cycles 100 -Scenario parent-death
```

| Scenario | What dies | Only possible mechanism |
|---|---|---|
| `stdin-eof` | the host | EOF on stdin |
| `parent-death` | the intermediate host, with the pipe held by a **survivor** | parent watch |
| `signal` | **nothing** | `CTRL_BREAK` |
| `daemon-idle` | the only bridge | idleness — the daemon has neither a parent nor a host stdin |

**Each scenario disconnects the others**, and the harness fails if the recorded reason is not the one belonging to the mechanism it names. Without that, falling through to the wrong mechanism would look green — which is how the parent watch crossed entire milestones without ever being exercised.

**A leaked orphan and an unmeasured cycle are different things.** A cycle that never launched the process observed nothing, neither success nor leak, and failing over it measures the machine's load. Up to 2% of cycles may go unmeasured — always printed, never silent — and `-MaxNaoMedidosPct 0` demands that every one of them measure. A real leak, or a run in which **no** cycle measured, fails at any threshold.

The script **does not build**: it refuses a `bin/gobsidian.exe` older than the code and tells you to run `build.ps1`. Without that guard, a stale binary passes the scenarios that do not depend on the new code and fails the ones that do, with a message pointing at the wrong place.

</details>

---

## 📚 Documentation

The documents below are written in Portuguese.

| Document | Contents |
|---|---|
| [`docs/PRD.md`](docs/PRD.md) | Problem, requirements, closed decisions, risks, milestones |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Layers, flows, data model, concurrency, lifecycle |
| [`docs/ESTRUTURA.md`](docs/ESTRUTURA.md) | Directory tree, each package's responsibility, conventions |
| [`docs/TOOLS.md`](docs/TOOLS.md) | Each tool's contract: input and output schemas, error codes |
| [`docs/WINDOWS.md`](docs/WINDOWS.md) | OneDrive, MAX_PATH, path casing, fsnotify |
| [`docs/OPERACAO.md`](docs/OPERACAO.md) | Measurements, the full table of non-functional requirements, diagnosis and known limits |
