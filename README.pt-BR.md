<div align="center">

# 🪨 gobsidian

**Servidor MCP de alto desempenho para cofres Obsidian locais.**  
Um binário Go. Zero dependências de runtime. Nenhum processo órfão.

🌍 **Português** · [English](README.md)

[![Go](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go&logoColor=white&style=flat-square)](https://go.dev)[![MCP](https://img.shields.io/badge/MCP-Standard-6E56CF?style=flat-square)](https://modelcontextprotocol.io)[![Plataformas](https://img.shields.io/badge/plataformas-Windows%20%7C%20macOS%20%7C%20Linux-informational?style=flat-square)](#-compatibilidade)[![Licença](https://img.shields.io/badge/licen%C3%A7a-MIT-green?style=flat-square)](LICENSE)

[Recursos](#-principais-recursos) • [Instalação](#-instalação) • [Configuração do host](#-configuração-do-host) • [Tools MCP](#-tools-mcp) • [Linha de comando](#-linha-de-comando) • [Compatibilidade](#-compatibilidade) • [Desenvolvimento](#-desenvolvimento) • [Documentação](#-documentação) • [Licença](#-licença)

---

</div>

O `gobsidian` liga seu cofre Obsidian local a qualquer cliente do Model Context Protocol (MCP) — Claude Desktop, Claude Code, Gemini CLI, Cursor, VS Code, Codex, Windsurf, Antigravity — sobre entrada e saída padrão (`stdio`).

---

## ✨ Principais recursos

* **Precisão abaixo do arquivo:** lê seções e blocos direto por offset de byte, sem carregar arquivos markdown grandes na memória.
* **Leitura de notas convertidas:** recupera a estrutura de títulos de notas convertidas de PDF, DOCX e EPUB.
* **Busca BM25 rápida:** motor de busca full-text em memória, com filtro por tag, pasta, data e frontmatter YAML.
* **Normalização multilíngue completa:** índice normalizado em NFC, com listas de postagem cruas e radicalizadas em paralelo (suporte pleno a acento).
* **Escrita atômica e segura:** edições dirigidas a seção (`note_append`, `note_patch`) passam por arquivo temporário e rename atômico, para não corromper o cofre.
* **Refatoração automática de links:** mover uma nota reescreve os `[[wikilinks]]`, âncoras e aliases que apontavam para ela.
* **Sem processos zumbis:** vários mecanismos de encerramento garantem que o processo termine quando o host fecha.
* **Confinamento estrito:** só I/O local; a flag `--read-only` remove da superfície MCP todo endpoint que escreve em disco.
* **Instala a si mesmo:** o binário **é** o instalador. Ele encerra instâncias em execução, limpa arquivos órfãos, se acrescenta ao `PATH` e escreve a configuração dos hosts MCP — sem privilégio de administrador e sem script à parte.

---

## 📦 Instalação

### Instalação rápida (automatizada)

```bash
# Windows (PowerShell):
iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install.ps1)

# macOS e Linux (bash):
curl -fsSL https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install.sh | sh

# Nushell
http get https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install.nu | nu --stdin -c $in
```

O script de bootstrap faz uma coisa só: baixa o binário para um diretório
temporário e roda `gobsidian install`. Todo o resto — encerrar instâncias
abertas, limpar arquivos de runtime órfãos, colocar o binário no diretório do
usuário (sem administrador), ajustar o `PATH` e configurar os hosts MCP
detectados — é trabalho do próprio binário, e tem teste.

<details>
<summary>⚙️ <b>Flags avançadas e métodos manuais</b></summary>

#### Flags

Toda flag depois do bootstrap é repassada literalmente para `gobsidian install`:

> **No nushell é o outro script.** O `install.nu` roda direto de um cano e não
> aceita flag: em modo `-c` o nushell consome os argumentos para si, e
> `nu --stdin -c $in --vault X` responde `Unknown flag '--vault'`. Salve o
> `install-flags.nu` num arquivo, e mantenha o `--` antes das suas flags — sem
> ele o nushell as lê como se fossem dele.

```bash
# Exemplo (Linux/macOS)
curl -fsSL .../install.sh | sh -s -- --vault "/caminho/do/cofre" --hosts claude-desktop --yes

# Exemplo (PowerShell)
& ([scriptblock]::Create((irm .../bootstrap/install.ps1))) --vault "C:\Meu Cofre" --hosts claude-desktop --yes

# Exemplo (Nushell) -- usa o install-flags.nu: ver a nota abaixo
http get https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install-flags.nu | save -f ($env.TMP | path join gobsidian-install.nu)
nu ($env.TMP | path join gobsidian-install.nu) -- --vault "C:\Meu Cofre" --hosts claude-desktop --yes
```

* `--vault <caminho>`: caminho do cofre (pula o menu interativo).
* `--hosts <lista>`: clientes alvo, separados por vírgula. Valores válidos:
  `antigravity`, `antigravity-ide`, `claude-code`, `claude-desktop`, `codex`,
  `cursor`, `gemini-cli`, `vscode`, `windsurf`. Use `none` para instalar o
  binário sem tocar em configuração de host nenhuma; omita a flag para detecção
  automática.
* `--read-only`: configura o servidor com as operações de escrita desabilitadas.
* `--install-dir <caminho>`: sobrepõe o diretório de instalação.
* `--no-path`: instala sem mexer no `PATH`.
* `--yes`: modo não interativo (assume as respostas padrão).

Arquivo de configuração de host nunca é sobrescrito: o `gobsidian` funde a
entrada dele e deixa o resto do seu JSON byte a byte, gravando um
`.gobsidian-backup` ao lado do arquivo antes de tocá-lo.

#### Binários pré-compilados

Baixe o binário da sua arquitetura em [Releases](https://github.com/jonyduque/Gobsidian/releases) e execute-o. Sem argumento nenhum ele se instala; `gobsidian install --help` lista as opções.

#### Compilar do código-fonte

Requer **Go 1.27+**:

```bash
git clone https://github.com/jonyduque/Gobsidian.git
cd Gobsidian
go build -o gobsidian ./cmd/gobsidian
```

> `go install github.com/...` **não** funciona: o caminho de módulo declarado no
> `go.mod` (`github.com/jonyd/gobsidian`) não é o caminho do repositório, então
> o proxy de módulos do Go não consegue resolvê-lo. Clone e compile.

</details>

---

## ⚙️ Configuração do host

O instalador configura os hosts detectados. Para (re)configurar um cofre depois,
sem reinstalar o binário:

```bash
gobsidian vaults --vault "/caminho/do/cofre"
```

Para registrar o `gobsidian` à mão:

### Registro por linha de comando
```bash
# Claude Code
claude mcp add gobsidian -- gobsidian serve --vault "/caminho/do/cofre"

# Gemini CLI
gemini mcp add gobsidian gobsidian serve --vault "/caminho/do/cofre"

# VS Code
code --add-mcp '{"name":"gobsidian","command":"gobsidian","args":["serve","--vault","/caminho/do/cofre"]}'
```

### Configuração em JSON (Claude Desktop, Cursor, Windsurf)
Acrescente ao arquivo de configuração MCP do seu cliente:

```json
{
  "mcpServers": {
    "gobsidian": {
      "command": "gobsidian",
      "args": ["serve", "--vault", "/caminho/absoluto/do/cofre"]
    }
  }
}
```
> **No Windows:** escape as barras invertidas no JSON (`"C:\\Users\\nome\\Cofre"`) ou use barras normais (`"C:/Users/nome/Cofre"`).

> **Para tirar o melhor das tools:** o schema MCP não consegue transmitir
> enumerações nem valores padrão ao modelo ([por quê](docs/TOOLS.md)), então há
> um prompt pronto em [`docs/PROMPT.md`](docs/PROMPT.md). Cole-o nas instruções
> do seu cliente.

---

## 🧰 Tools MCP

O contrato de schema e a matriz de erros de cada tool estão em [`docs/TOOLS.md`](docs/TOOLS.md).

### Leitura
| Tool | Descrição |
|---|---|
| `vault_search` | Busca BM25, com correspondência exata e filtros de frontmatter e tag. |
| `note_read` | Lê a nota inteira, um `# título` específico ou um alvo `^block-id`. Aceita lote. |
| `note_outline` | Devolve a hierarquia estrutural, os títulos explícitos e os candidatos a título. |
| `note_list` | Filtra notas por pasta, glob, tag ou metadado YAML. |
| `note_metadata` | Extrai frontmatter YAML, links de saída, backlinks e tags. |
| `link_graph` | Percorre a vizinhança do grafo, com profundidade e direção configuráveis. |
| `vault_broken_links` | Relata wikilinks mortos e âncoras soltas, com o contexto de origem. |
| `tag_list` | Varre as tags do cofre e devolve a contagem de ocorrências. |
| `vault_stats` | Total de notas, arquivos órfãos, links quebrados e contadores do watcher. |

### Escrita
*Toda escrita aceita `dry_run` e proteção contra concorrência via `expected_hash`.*

| Tool | Descrição |
|---|---|
| `note_create` | Cria uma nota (falha em segurança se o arquivo já existe). |
| `note_append` | Acrescenta conteúdo ao fim da nota ou logo abaixo de um título. |
| `note_patch` | Substitui o conteúdo sob um título ou block ID, atomicamente. |
| `note_move` | Renomeia ou move o arquivo e atualiza os `[[wikilinks]]` que apontavam para ele. |
| `note_delete` | Remove o arquivo depois de apresentar o relatório de impacto dos links que quebrariam. |

---

## 💻 Linha de comando

O `gobsidian` traz ferramentas de linha de comando para instalação, diagnóstico, teste e indexação fora de qualquer host MCP:

```bash
# Sobe o servidor MCP sobre stdio
gobsidian serve --vault "/caminho/do/cofre" [--read-only]

# Instala ou reconfigura (sem argumento nenhum também instala)
gobsidian install [--vault <caminho>] [--hosts <lista>] [--read-only] [--yes]

# Atualiza para a última versão publicada
gobsidian update [--check] [--yes]

# Configura os hosts para um cofre, sem reinstalar o binário
gobsidian vaults --vault "/caminho/do/cofre"

# Acrescenta ou remove o diretório de instalação do PATH do usuário
gobsidian path [--add|--remove]

# Diagnóstico do ambiente e do cofre (permissão, colisão de caixa, MAX_PATH)
gobsidian doctor --vault "/caminho/do/cofre"

# Constrói o cache de índice
gobsidian index --vault "/caminho/do/cofre" [--json]

# Busca pela linha de comando
gobsidian search "consulta" --vault "/caminho/do/cofre" [--limit 20]

# Inspeciona a representação analisada de um arquivo
gobsidian inspect "Nota.md" --vault "/caminho/do/cofre" [--json]

# Imprime versão, commit e data de build
gobsidian version
```

### Completação de shell

A completação cobre nome de comando, nome de flag **e valor de flag** —
`--vault` oferece os cofres que o Obsidian conhece, marcando os que estão
abertos; `--hosts`, as nove chaves com o nome de cada produto; `--log-level`,
os quatro níveis.

```bash
# bash / zsh / fish / powershell
gobsidian completion bash > /etc/bash_completion.d/gobsidian

# nushell — acrescente ao seu config.nu
let gobsidian_completer = {|spans| gobsidian _carapace nushell ...$spans | from json }
$env.config.completions.external = {
  enable: true
  completer: $gobsidian_completer
}
```

`gobsidian _carapace <shell>` cobre também elvish, oil, tcsh e xonsh.

---

## 🖥️ Compatibilidade

| Item | Suporte |
|---|---|
| **Sistemas operacionais** | Windows, macOS e Linux — um binário estático por plataforma, sem dependência de runtime. |
| **Go (compilando da fonte)** | 1.27 ou mais novo. Os binários publicados não exigem nada instalado. |
| **Hosts MCP** | Claude Desktop, Claude Code, Gemini CLI, Cursor, VS Code, Codex, Windsurf e Antigravity (IDE e avulso). |
| **Armazenamento do cofre** | Disco local, inclusive pasta sincronizada pelo OneDrive. Arquivo somente-nuvem é indexado pelo nome e nunca aberto, então nenhum download é disparado. |
| **Anexos** | Indexados pelo nome; o conteúdo nunca é lido. |

O Windows é o alvo principal e tem a maior cobertura: caminho longo, colisão de
caixa, placeholder do OneDrive e as peculiaridades do `fsnotify` estão
documentados em [`docs/WINDOWS.md`](docs/WINDOWS.md).

Quando vários hosts MCP abrem o mesmo cofre, eles compartilham um único daemon
em segundo plano por socket local, em vez de cada cliente construir o próprio
índice. O encerramento é coberto por quatro mecanismos (EOF do stdin, sinal,
morte do processo pai e ociosidade), e cada um é exercitado por um gate de 100
ciclos a cada rodada de CI — é isso que "nenhum processo órfão" quer dizer aqui.

---

## 🛠️ Desenvolvimento

```bash
# Bateria de validação (build, race, lint, vet nos três GOOS, gates)
pwsh -File scripts/verify.ps1

# Compila o binário de release
pwsh -File scripts/build.ps1

# Gate de encerramento sem processo órfão
pwsh -File scripts/test_orphans.ps1 -Cycles 100
```

`verify.ps1` verde é obrigatório antes de qualquer commit. As convenções de
contribuição, as regras de arquitetura e o registro histórico de todo defeito
que já custou caro aqui estão em [`CLAUDE.md`](CLAUDE.md) e em
[`docs/ARMADILHAS.md`](docs/ARMADILHAS.md).

---

## 📚 Documentação

* [`docs/PRD.md`](docs/PRD.md) — Escopo, objetivos de projeto e requisitos não funcionais.
* [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — Arquitetura, concorrência e cache.
* [`docs/ESTRUTURA.md`](docs/ESTRUTURA.md) — Estrutura do código e responsabilidade de cada pacote.
* [`docs/TOOLS.md`](docs/TOOLS.md) — Schemas, entradas e matriz de erros de cada tool MCP.
* [`docs/PROMPT.md`](docs/PROMPT.md) — Prompt pronto para clientes MCP, e por que ele é necessário.
* [`docs/WINDOWS.md`](docs/WINDOWS.md) — Peculiaridades do Windows (OneDrive, fsnotify, caminho longo).
* [`docs/OPERACAO.md`](docs/OPERACAO.md) — Diagnóstico, medições de latência e limites operacionais.

---

## 📄 Licença

MIT © [Jony Duque](LICENSE)
