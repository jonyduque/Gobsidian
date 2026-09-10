<div align="center">

# 🪨 gobsidian

**High-performance MCP server for local Obsidian vaults.**  
Single Go binary. Zero runtime dependencies. No orphan processes.

🌍 **English** · [Português](README.pt-BR.md)

[![Go](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go&logoColor=white&style=flat-square)](https://go.dev)[![MCP](https://img.shields.io/badge/MCP-Standard-6E56CF?style=flat-square)](https://modelcontextprotocol.io)[![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20macOS%20%7C%20Linux-informational?style=flat-square)](#-compatibility)[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)

[Features](#-key-features) • [Installation](#-installation) • [Host Configuration](#-host-configuration) • [MCP Tools](#-mcp-tools) • [CLI Reference](#-cli-reference) • [Compatibility](#-compatibility) • [Development](#-development) • [Docs](#-documentation) • [License](#-license)

---

</div>

`gobsidian` connects your local Obsidian vault to any Model Context Protocol (MCP) client (Claude Desktop, Claude Code, Gemini CLI, Cursor, VS Code, Codex, Windsurf, Antigravity) over standard input/output (`stdio`).

---

## ✨ Key Features

* **Sub-file Precision:** Reads sections and blocks directly by byte offsets without loading massive markdown files into memory.
* **Smart Converted Note Parsing:** Recovers heading structures from converted PDF, DOCX, and EPUB notes.
* **Fast BM25 Search:** In-memory full-text search engine with tag, folder, date, and YAML frontmatter filtering.
* **Full Multilingual Normalization:** NFC-normalized index with dual raw/stemmed posting lists (full Portuguese accent support).
* **Atomic & Safe Writes:** Section-targeted edits (`note_append`, `note_patch`) run through temp files and atomic renames to prevent vault corruption.
* **Automatic Link Refactoring:** Moving notes automatically rewrites incoming `[[wikilinks]]`, anchors, and aliases.
* **Zombie Prevention:** Multi-tier shutdown mechanisms guarantee processes terminate cleanly when client hosts close.
* **Strict Confinement:** Runs in pure local I/O mode; `--read-only` flag strips all disk-mutation endpoints from MCP exposure.
* **Self-installing:** The binary *is* the installer. It closes running instances, cleans stale files, adds itself to `PATH`, and writes the MCP host configuration — no admin rights, no separate script.

---

## 📦 Installation

### Quick Install (Automated)

```bash
# Windows (PowerShell):
iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install.ps1)

# macOS & Linux (bash):
curl -fsSL https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install.sh | sh

# Nushell
http get https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install.nu | nu --stdin -c $in
```

The bootstrap script does one thing: it downloads the binary to a temporary
directory and runs `gobsidian install`. Everything else — closing running
instances, cleaning stale runtime files, placing the binary in your user
directory (no admin/root needed), updating `PATH`, and configuring detected
MCP hosts — is the binary's own work, and is covered by tests.

<details>
<summary>⚙️ <b>Advanced Install Flags & Manual Methods</b></summary>

#### Automated Flags

Every flag after the bootstrap is forwarded verbatim to `gobsidian install`:

> **Nushell needs the other script.** `install.nu` runs straight from a pipe and
> takes no flags: in `-c` mode nushell consumes the arguments itself, and
> `nu --stdin -c $in --vault X` answers `Unknown flag '--vault'`. Save
> `install-flags.nu` to a file instead, and keep the `--` before your flags —
> without it nushell reads them as its own.

```bash
# Example (Linux/macOS)
curl -fsSL .../install.sh | sh -s -- --vault "/path/to/vault" --hosts claude-desktop --yes

# Example (PowerShell)
& ([scriptblock]::Create((irm .../bootstrap/install.ps1))) --vault "C:\My Vault" --hosts claude-desktop --yes

# Example (Nushell) -- uses install-flags.nu: see the note below
http get https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install-flags.nu | save -f ($env.TMP | path join gobsidian-install.nu)
nu ($env.TMP | path join gobsidian-install.nu) -- --vault "C:\My Vault" --hosts claude-desktop --yes
```

* `--vault <path>`: Direct vault path (skips interactive menu).
* `--hosts <list>`: Comma-separated target clients. Valid values: `antigravity`,
  `antigravity-ide`, `claude-code`, `claude-desktop`, `codex`, `cursor`,
  `gemini-cli`, `vscode`, `windsurf`. Use `none` to install the binary without
  touching any host configuration; omit the flag to auto-detect.
* `--read-only`: Sets up the server with write operations disabled.
* `--install-dir <path>`: Overrides the installation directory.
* `--no-path`: Installs without touching `PATH`.
* `--yes`: Non-interactive mode (assumes default answers).

Host configuration files are never overwritten: `gobsidian` merges its entry
and leaves the rest of your JSON byte-for-byte, writing a `.gobsidian-backup`
next to the file before touching it.

#### Precompiled Binaries

Download the binary for your architecture from [Releases](https://github.com/jonyduque/Gobsidian/releases) and run it. With no arguments it installs itself; `gobsidian install --help` lists the options.

#### Build from Source

Requires **Go 1.27+**:

```bash
git clone https://github.com/jonyduque/Gobsidian.git
cd Gobsidian
go build -o gobsidian ./cmd/gobsidian
```

> `go install github.com/...` does **not** work: the module path declared in
> `go.mod` (`github.com/jonyd/gobsidian`) is not the repository path, so the
> Go module proxy cannot resolve it. Clone and build.

</details>

---

## ⚙️ Host Configuration

The installer configures detected hosts for you. To (re)configure a vault later
without reinstalling the binary:

```bash
gobsidian vaults --vault "/path/to/vault"
```

To register `gobsidian` by hand instead:

### CLI Registrations
```bash
# Claude Code
claude mcp add gobsidian -- gobsidian serve --vault "/path/to/vault"

# Gemini CLI
gemini mcp add gobsidian gobsidian serve --vault "/path/to/vault"

# VS Code
code --add-mcp '{"name":"gobsidian","command":"gobsidian","args":["serve","--vault","/path/to/vault"]}'
```

### JSON Configuration (Claude Desktop, Cursor, Windsurf)
Add to your client's MCP configuration file:

```json
{
  "mcpServers": {
    "gobsidian": {
      "command": "gobsidian",
      "args": ["serve", "--vault", "/absolute/path/to/vault"]
    }
  }
}
```
> **Windows Tip:** Escape backslashes in JSON configuration files (`"C:\\Users\\name\\Vault"`) or use forward slashes (`"C:/Users/name/Vault"`).

> **Getting the best out of the tools:** the MCP schema cannot carry enumerations
> or defaults to the model ([why](docs/TOOLS.md)), so a ready-made prompt is
> provided in [`docs/PROMPT.md`](docs/PROMPT.md). Paste it into your client's
> instructions.

---

## 🧰 MCP Tools

Complete schema contracts and error definitions are detailed in [`docs/TOOLS.md`](docs/TOOLS.md).

### Read Operations
| Tool | Description |
|---|---|
| `vault_search` | Search via BM25 query with exact matching and frontmatter/tag filters. |
| `note_read` | Read full notes, specific `# headings`, or `^block-id` targets. Supports batching. |
| `note_outline` | Retrieve structural hierarchy, explicit headings, and synthetic candidate titles. |
| `note_list` | Filter notes by folder, glob patterns, tags, or YAML metadata. |
| `note_metadata` | Extract YAML frontmatter, outgoing links, backlinks, and tags. |
| `link_graph` | Graph neighborhood traversal with configurable depth and direction. |
| `vault_broken_links` | Report dead wikilinks and dangling anchors with source context. |
| `tag_list` | Scan all vault tags and return aggregated occurrence counts. |
| `vault_stats` | Inspect total notes, orphaned files, broken links, and watcher counters. |

### Write Operations
*All write actions support `dry_run` and concurrency protection via `expected_hash`.*

| Tool | Description |
|---|---|
| `note_create` | Create a new note file (safely fails if file exists). |
| `note_append` | Append content to note bottom or directly under a specific heading. |
| `note_patch` | Replace contents under a specific heading or block ID atomically. |
| `note_move` | Rename/relocate files and update all incoming `[[wikilinks]]` across the vault. |
| `note_delete` | Remove file after surfacing an impact report of newly broken links. |

---

## 💻 CLI Reference

`gobsidian` includes standalone CLI tools for installation, diagnostics, testing, and indexing outside of MCP hosts:

```bash
# Start MCP server over stdio
gobsidian serve --vault "/path/to/vault" [--read-only]

# Install or reconfigure (no arguments at all also installs)
gobsidian install [--vault <path>] [--hosts <list>] [--read-only] [--yes]

# Update to the latest published release
gobsidian update [--check] [--yes]

# Configure hosts for a vault, without reinstalling the binary
gobsidian vaults --vault "/path/to/vault"

# Add or remove the install directory from your user PATH
gobsidian path [--add|--remove]

# Run system and vault health check (permissions, case collisions, MAX_PATH)
gobsidian doctor --vault "/path/to/vault"

# Build index cache directly
gobsidian index --vault "/path/to/vault" [--json]

# Execute CLI search
gobsidian search "query" --vault "/path/to/vault" [--limit 20]

# Inspect parsed representation of a file
gobsidian inspect "Note.md" --vault "/path/to/vault" [--json]

# Print version, commit and build date
gobsidian version
```

### Shell completion

Completion covers command names, flag names **and flag values** — `--vault`
offers the vaults Obsidian knows about (marking the ones currently open),
`--hosts` the nine host keys with their product names, `--log-level` the four
levels.

```bash
# bash / zsh / fish / powershell
gobsidian completion bash > /etc/bash_completion.d/gobsidian

# nushell — add to your config.nu
let gobsidian_completer = {|spans| gobsidian _carapace nushell ...$spans | from json }
$env.config.completions.external = {
  enable: true
  completer: $gobsidian_completer
}
```

`gobsidian _carapace <shell>` also covers elvish, oil, tcsh and xonsh.

---

## 🖥️ Compatibility

| Item | Support |
|---|---|
| **Operating systems** | Windows, macOS, Linux — a single static binary per platform, no runtime dependencies. |
| **Go (build from source)** | 1.27 or newer. The published binaries need nothing installed. |
| **MCP hosts** | Claude Desktop, Claude Code, Gemini CLI, Cursor, VS Code, Codex, Windsurf, Antigravity (IDE and standalone). |
| **Vault storage** | Local disk, including OneDrive-synced folders. Cloud-only placeholders are indexed by name and never opened, so no download is triggered. |
| **Attachments** | Indexed by filename; their contents are never read. |

Windows is the primary target and gets the most coverage: long paths, case
collisions, OneDrive placeholders, and `fsnotify` edge cases are documented in
[`docs/WINDOWS.md`](docs/WINDOWS.md).

When several MCP hosts open the same vault, they share one background daemon
over a local socket instead of building one index per client. Shutdown is
covered by four mechanisms (stdin EOF, signal, parent death, idle timeout), each
exercised by a 100-cycle gate on every CI run — that is what "no orphan
processes" means here.

---

## 🛠️ Development

```bash
# Run validation battery (build, race, lint, vet across three GOOS, gates)
pwsh -File scripts/verify.ps1

# Build release binary
pwsh -File scripts/build.ps1

# Run orphan-process termination gate
pwsh -File scripts/test_orphans.ps1 -Cycles 100
```

`verify.ps1` green is required before any commit. Contribution conventions,
architectural rules, and the historical record of every defect that cost time
here live in [`CLAUDE.md`](CLAUDE.md) and [`docs/ARMADILHAS.md`](docs/ARMADILHAS.md).

---

## 📚 Documentation

Detailed documentation (written in Portuguese):

* [`docs/PRD.md`](docs/PRD.md) — Scope, design goals, non-functional requirements.
* [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — System architecture, concurrency, and caching.
* [`docs/ESTRUTURA.md`](docs/ESTRUTURA.md) — Codebase structure and package responsibilities.
* [`docs/TOOLS.md`](docs/TOOLS.md) — Complete schemas, inputs, and error matrices for MCP tools.
* [`docs/PROMPT.md`](docs/PROMPT.md) — Ready-made prompt for MCP clients, and why it is needed.
* [`docs/WINDOWS.md`](docs/WINDOWS.md) — Windows edge cases (OneDrive, fsnotify, long paths).
* [`docs/OPERACAO.md`](docs/OPERACAO.md) — Diagnostics, latency benchmarks, and operational limits.

---

## 📄 License

MIT © [Jony Duque](LICENSE)
