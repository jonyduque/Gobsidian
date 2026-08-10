<div align="center">

# 🪨 gobsidian

**Servidor MCP para cofres locais do Obsidian. Um binário Go, sem runtime, sem processo órfão.**

[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2025--11--25-6E56CF)](https://modelcontextprotocol.io)
[![Plataformas](https://img.shields.io/badge/plataformas-Windows%20%7C%20Linux%20%7C%20macOS-informational)](#-compatibilidade)
[![Licença](https://img.shields.io/badge/licen%C3%A7a-MIT-green)](LICENSE)

[Visão geral](#-visão-geral) •
[Instalação](#-instalação) •
[Configuração](#-configuração-no-host-mcp) •
[Ferramentas](#-ferramentas-mcp) •
[CLI](#-linha-de-comando) •
[Desempenho](#-desempenho) •
[Documentação](#-documentação)

</div>

---

## 🔍 Visão geral

`gobsidian` expõe um cofre local do Obsidian a qualquer host MCP — Claude Desktop, Claude Code, Gemini CLI, Antigravity, VS Code — através de um executável único que fala JSON-RPC sobre stdio.

O projeto nasceu de três problemas concretos dos servidores MCP de Obsidian existentes:

| 🐛 Problema | ✅ O que `gobsidian` faz |
|---|---|
| Processos zumbi depois que o host fecha | Três mecanismos independentes de encerramento, verificados em 100 ciclos de morte abrupta cada |
| Reindexação total a cada evento de arquivo | Watcher incremental com coalescência, e índice por deslocamento de byte |
| Parsers genéricos que quebram em wikilink, embed e block-id | Parser próprio, congelado por 48 arquivos golden e conferido contra um dump real do `metadataCache` do Obsidian |

### ✨ Funcionalidades

- ⚡ **Leitura por offset.** O índice guarda o deslocamento de byte de cada heading e bloco. Ler uma seção de 2 KB numa nota de 500 KB custa 2 KB de I/O, não 500 KB.
- 🔎 **Busca full-text com BM25**, pesos por campo (título 3x, headings 2x, corpo 1x), busca por frase exata e filtros combináveis de pasta, tag, frontmatter e data.
- 🇧🇷 **Analisador para português**: normalização de acentos, *case folding* e indexação dupla — a forma crua e a reduzida na mesma posting list, para que o recall venha do stem e a precisão venha do termo original.
- ✍️ **Escrita atômica e cirúrgica.** `note_append` e `note_patch` operam por heading ou por block-id: inserção delta em posição conhecida, não reescrita do arquivo inteiro. Toda escrita passa por temporário + rename, para que o Obsidian aberto ao lado nunca veja um arquivo pela metade.
- 🔗 **`note_move` reescreve os links.** Alias, âncora e forma original preservados; falha parcial reporta exatamente o que foi aplicado.
- 👀 **Watcher incremental** com debounce, conjunto sujo e reconciliação por varredura quando o `fsnotify` estoura.
- 🔒 **Modo `--read-only`** que remove as tools de escrita de `ListTools` — ausentes, não apenas rejeitadas.
- 🚫 **Zero rede.** Nenhum pacote sob `internal/` ou `cmd/` importa `net`, e isso é cobrado no CI por um analisador de `go vet` nos três sistemas.

> [!NOTE]
> O cofre é a fonte da verdade; o índice é derivado e descartável. Se corromper, reconstrói em segundos. `gobsidian` nunca mantém estado que exista só em memória.

---

## 📦 Instalação

Qualquer um dos instaladores abaixo faz a mesma coisa: baixa o binário da release, **confere o SHA-256 contra o `SHA256SUMS.txt` publicado junto e aborta se não bater**, instala, acrescenta ao `PATH`, lê o registro de cofres do próprio Obsidian para você não ter de ir atrás do caminho, e pergunta em quais hosts de IA da máquina registrar o servidor.

| Você está em | Use | Precisa de |
|---|---|---|
| 🪟 Windows | `install.ps1` da raiz | nada além do PowerShell |
| 🪟🐧🍎 Windows, Linux ou macOS | `installer/` | Node.js 18+ |

### 🚀 Windows, sem dependência nenhuma

```powershell
iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/install.ps1)
```

### 🌍 Multiplataforma — pasta `installer/`

O mesmo instalador escrito em Node, para quem está em Linux ou macOS, ou prefere um só caminho para as três plataformas. Requer **Node.js 18+** e nada mais: `install.js` importa apenas módulos nativos do Node, então não há `npm install`, `node_modules` nem dependência de terceiros na sua máquina.

**Windows:**

```powershell
iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.ps1)
```

**Linux e macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.bash | bash
```

<details>
<summary>⚙️ <b>Opções do instalador</b> — instalação silenciosa, host específico, versão fixa</summary>

<br>

`iex` executa o **texto** do script, e um bloco `param()` não recebe nada por essa via. Para passar opções ao instalador da raiz, crie um scriptblock:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/install.ps1))) `
    -Vault "C:\Meu Cofre" -Hosts claude-desktop,claude-code -ReadOnly
```

O da pasta `installer/` recebe as opções no estilo de linha de comando, com dois hífens, e repassa o que vier depois:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.ps1))) `
    --vault "C:\Meu Cofre" --hosts claude-desktop,claude-code --read-only
```

```bash
curl -fsSL https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.bash | bash -s -- \
    --vault "/caminho/do/cofre" --hosts claude-desktop,claude-code --yes
```

| Raiz | `installer/` | Efeito |
|---|---|---|
| `-Vault <caminho>` | `--vault <caminho>` | Cofre a servir. Repetível, ou vários separados por vírgula. Sem isto, o instalador procura e pergunta. |
| `-Hosts <chaves>` | `--hosts <chaves>` | Configura só esses hosts, sem menu. |
| `-Version <tag>` | `--version <tag>` | Instala uma release específica. Padrão: a mais recente. |
| `-InstallDir <caminho>` | `--install-dir <caminho>` | Onde por o binário. |
| `-ReadOnly` | `--read-only` | Registra o servidor com `--read-only`. |
| `-Yes` | `--yes`, `-y` | Não pergunta nada. Exige o cofre por opção ou variável. |
| `-NoPath` | `--no-path` | Não mexe no `PATH`. |
| — | `--force` | Reinstala mesmo que a versão já esteja presente. |

Chaves aceitas: `claude-desktop`, `claude-code`, `gemini-cli`, `antigravity`, `antigravity-ide`, `codex`, `vscode`, `cursor`, `windsurf`.

As mesmas opções funcionam por variável de ambiente, nos dois instaladores, e atravessam todas as formas de invocação:

```powershell
$env:GOBSIDIAN_VAULT = "C:\Meu Cofre"
iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.ps1)
```

`GOBSIDIAN_VAULT`, `GOBSIDIAN_VERSION` e `GOBSIDIAN_INSTALL_DIR`.

</details>

> [!TIP]
> Com elevação, a instalação vai para `Program Files` e o `PATH` de máquina. Sem elevação, vai para `%LOCALAPPDATA%\Programs\gobsidian` e o `PATH` de usuário — o instalador **não** pede UAC por conta própria.

> [!IMPORTANT]
> Os dois instaladores abortam se o `SHA256SUMS.txt` da release estiver ausente ou se a soma não bater. Um binário que vai para o `PATH` sem verificação é um buraco de cadeia de suprimentos com aparência de conveniência.

> [!NOTE]
> **Atualizando de uma v1.0.x?** O formato do cache de busca mudou, então a primeira partida reconstrói o índice do cofre em segundo plano — as ferramentas respondem desde o primeiro segundo, mas a busca só entra quando a reconstrução termina. Se você mantiver as duas versões instaladas e alternar entre elas, cada uma vai invalidar o cache da outra indefinidamente: instale por cima, não ao lado.

### 📥 Binário pré-compilado

Baixe o executável do seu sistema em [Releases](https://github.com/jonyduque/Gobsidian/releases) e coloque em qualquer diretório do `PATH`. Não há serviço, não há registro no sistema, não há nada para desinstalar além do arquivo.

| Sistema | Arquivo |
|---|---|
| 🪟 Windows x86-64 | `gobsidian-windows-amd64.exe` |
| 🐧 Linux x86-64 | `gobsidian-linux-amd64` |
| 🍎 macOS Apple Silicon | `gobsidian-darwin-arm64` |

Confira o download contra o `SHA256SUMS.txt` publicado junto:

```powershell
(Get-FileHash .\gobsidian-windows-amd64.exe -Algorithm SHA256).Hash
```

```bash
sha256sum -c SHA256SUMS.txt --ignore-missing
```

### 🔨 A partir do código-fonte

Requer **Go 1.25 ou superior** — o piso vem do SDK oficial de MCP, não é preferência.

```bash
go install github.com/jonyd/gobsidian/cmd/gobsidian@latest
gobsidian version
```

---

## ⚙️ Configuração no host MCP

O instalador faz isto sozinho. As seções abaixo são para quem prefere configurar à mão.

<details>
<summary>📝 <b>Configuração manual por host</b> — Claude Desktop, Claude Code, Gemini CLI, Antigravity, VS Code</summary>

<br>

**Claude Desktop** — `%APPDATA%\Claude\claude_desktop_config.json` no Windows, `~/Library/Application Support/Claude/claude_desktop_config.json` no macOS. Acrescente a chave dentro de `mcpServers`, preservando o que já estiver lá:

```json
{
  "mcpServers": {
    "gobsidian": {
      "command": "C:\\Program Files\\gobsidian\\gobsidian.exe",
      "args": ["serve", "--vault", "C:\\Meu Cofre"]
    }
  }
}
```

**Claude Code** e **Gemini CLI** registram sozinhos — repare que a ordem dos argumentos difere entre os dois:

```bash
claude mcp add gobsidian --scope user -- gobsidian serve --vault "C:\Meu Cofre"
gemini mcp add gobsidian gobsidian serve --vault "C:\Meu Cofre" --scope user
```

**VS Code**:

```bash
code --add-mcp '{"name":"gobsidian","command":"gobsidian","args":["serve","--vault","C:\\Meu Cofre"]}'
```

**Antigravity** e **Antigravity IDE** usam o mesmo formato do Claude Desktop, em `~/.gemini/antigravity/mcp_config.json` e `~/.gemini/antigravity-ide/mcp_config.json`.

**Cursor** usa `~/.cursor/mcp.json`; **Windsurf**, `~/.codeium/windsurf/mcp_config.json`. Mesmo formato.

> [!IMPORTANT]
> Três erros respondem pela maioria das falhas de configuração no Windows:
>
> 1. **Barra invertida simples.** JSON exige escape: `\\` em cada separador.
> 2. **Aspas a mais em volta de caminho com espaço.** Cada elemento de `args` já é uma string; aspas internas viram parte do caminho, e o servidor recebe algo que não existe.
> 3. **Caminho relativo para o binário.** O host não herda necessariamente o `PATH` do seu shell. Use caminho absoluto.

</details>

<details>
<summary>🎛️ <b>Opções de <code>serve</code></b></summary>

<br>

| Flag | Efeito |
|---|---|
| `--vault <caminho>` | Raiz do cofre. Obrigatória. |
| `--read-only` | Remove toda a superfície de escrita. |
| `--cache-dir <caminho>` | Diretório do cache de índice. Padrão: derivado de um hash do caminho do cofre, sempre **fora** do cofre. |
| `--debounce-ms <n>` | Janela de coalescência de eventos do watcher. |
| `--log-level <nível>` | `debug`, `info`, `warn` ou `error`. |
| `--eager-search` | Carrega o índice de busca no boot, em vez de esperar a primeira `vault_search`. Padrão: preguiçoso — a maioria das sessões de assistente lê e escreve nota sem nunca buscar, e pagar o índice de qualquer jeito custa RSS que a sessão nunca usa. |

</details>

> [!TIP]
> Comece com `--read-only` até confiar na configuração. Você mantém busca, leitura e grafo de links, sem nenhuma tool capaz de tocar no disco.

---

## 🧰 Ferramentas MCP

Contratos completos, schemas e códigos de erro em [`docs/TOOLS.md`](docs/TOOLS.md).

### 📖 Leitura

| Tool | O que faz |
|---|---|
| `vault_search` | Busca full-text com ranking BM25, frase exata e filtros de pasta, tag, frontmatter e data |
| `note_read` | Nota inteira, uma seção por heading, ou um bloco por `^id` |
| `note_list` | Lista por glob, pasta, tag ou consulta de frontmatter |
| `note_metadata` | Frontmatter, tags, links de saída, backlinks, headings e blocos |
| `link_graph` | Vizinhança de links, com direção e profundidade configuráveis |
| `tag_list` | Todas as tags do cofre, com contagem |
| `vault_stats` | Notas, órfãs, links quebrados, âncoras quebradas e contadores do watcher |

### ✏️ Escrita

| Tool | O que faz |
|---|---|
| `note_create` | Cria a nota, com frontmatter opcional, falhando se já existir |
| `note_append` | Anexa ao fim da nota ou ao fim de uma seção |
| `note_patch` | Substitui o conteúdo sob um heading ou de um bloco |
| `note_move` | Move ou renomeia, reescrevendo todos os wikilinks que apontam para a nota |
| `note_delete` | Remove, com relatório prévio dos links que vão quebrar |

Todas as tools de escrita aceitam `dry_run` e `expected_hash`.

<details>
<summary>🗂️ <b>Resources</b> — notas anexadas ao contexto sem chamada de tool</summary>

<br>

As notas são publicadas como resources MCP sob `gobsidian:///<caminho>`, com escape percent.

O esquema é próprio de propósito: `obsidian://` pertence ao aplicativo Obsidian e é registrado no sistema operacional.

As **três** barras também são deliberadas. Em RFC 3986, o que vem depois de `//` é a **autoridade**, não o caminho — com duas barras, `Pasta/Minha nota.md` faria `Pasta` virar nome de host, e uma nota na raiz com espaço no nome tornaria o host inteiro inválido. A terceira barra declara autoridade vazia e faz o caminho começar onde deve.

</details>

---

## 💻 Linha de comando

`gobsidian` funciona fora do MCP, o que é o que torna o diagnóstico possível.

```bash
# Servir via stdio (é o que o host MCP executa)
gobsidian serve --vault "/caminho/do/cofre"

# Diagnosticar o ambiente antes de qualquer outra coisa
gobsidian doctor --vault "/caminho/do/cofre"

# Indexar e sair, com um resumo do cofre
gobsidian index --vault "/caminho/do/cofre" --json

# Buscar direto, sem host MCP
gobsidian search "arquitetura de software" --vault "/caminho/do/cofre" --limit 10

# Ver como uma nota foi interpretada
gobsidian inspect "Pasta/Minha nota.md" --vault "/caminho/do/cofre" --json
```

> [!TIP]
> 🩺 `gobsidian doctor` é o primeiro comando a rodar quando algo não funciona. Ele verifica o que costuma quebrar antes de tudo: cofre inacessível, permissões, arquivos OneDrive somente-nuvem, caminhos acima de 260 caracteres e colisões de caixa entre pastas.

`doctor` e `version` escrevem em **stdout** de propósito, porque são comandos de CLI. Em `serve`, stdout pertence inteiro ao JSON-RPC e todo log vai para stderr.

---

## 🧵 Daemon (opcional, ligado por padrão)

Cada sessão de host MCP abre um processo `gobsidian serve`. Se duas ou mais sessões apontam para o **mesmo cofre ao mesmo tempo** e alguma delas de fato usa `vault_search`, o daemon compartilha um único índice entre elas em vez de cada sessão carregar a própria cópia — a ponte de cada sessão fica com poucos MB e o custo pesado é pago uma vez só. Para uma sessão isolada, ou para sessões que só leem e escrevem nota sem nunca buscar, o daemon não muda nada: o custo dele é essencialmente igual ao de rodar sem ele. Números completos em [`docs/OPERACAO.md`](docs/OPERACAO.md), seção "Parte II do M7".

O transporte é um socket de domínio Unix local, sob o diretório de runtime do usuário — nunca um socket que sai da máquina (RNF-30, `docs/PRD.md` §6.4).

**Desligar o daemon.** A variável de ambiente `GOBSIDIAN_NO_DAEMON=1` força todo `gobsidian serve` a rodar sempre em processo, sem tentar discar ou iniciar um daemon:

```bash
GOBSIDIAN_NO_DAEMON=1 gobsidian serve --vault "/caminho/do/cofre"
```

No `claude_desktop_config.json` (ou equivalente do host), a variável entra no bloco `env` da entrada do servidor.

**O que acontece quando ele não sobe.** O fallback é automático e silencioso para quem usa a ferramenta: se a discagem ao socket falhar (daemon ausente, socket quebrado, versão ou configuração incompatível — por exemplo `--read-only` divergente de uma instância já conectada) e a tentativa seguinte de iniciar um daemon também falhar, `gobsidian serve` cai para o modo em processo de sempre, o mesmo caminho usado quando o daemon está desligado. Nenhum caminho deixa a ferramenta inutilizável por causa do daemon; o pior caso é perder o compartilhamento de memória, não a funcionalidade. O daemon em si sai sozinho depois de 15 minutos sem nenhuma sessão conectada.

---

## 🖥️ Compatibilidade

| Item | Estado |
|---|---|
| 🪟 Windows 10+ | Plataforma de primeira classe |
| 🐧 Linux (kernel 5.x+) e 🍎 macOS 13+ | Suportados; o CI roda build, `vet` e `go test -race` nos três |
| Protocolo MCP | `2025-11-25`. A negociação para versões anteriores (`2025-06-18`, `2025-03-26`, `2024-11-05`) é do SDK oficial, fixado em `v1.5.0` |
| ☁️ Cofres em OneDrive, Dropbox e Google Drive | Suportados, incluindo arquivos somente-nuvem, que nunca são abertos para não disparar download |
| Caminhos acima de 260 caracteres no Windows | Suportados |

Hosts que o instalador reconhece: Claude Desktop, Claude Code, Gemini CLI, Antigravity, Antigravity IDE, Codex CLI, VS Code, Cursor e Windsurf.

---

## 📊 Desempenho

Medido em cofre sintético determinístico de **5.000 notas** (`scripts/gen_vault.ps1 -Notes 5000 -Seed 42`), num notebook de 12 núcleos com Windows 11, sem `-race`.

| Requisito | Alvo | Medido | |
|---|---|---|---|
| Indexação a frio (metadados) | ≤ 3 s | **500 ms**; 1,1 s num cofre de 109 MB | ✅ |
| Boot com cache válido | ≤ 300 ms | 208–282 ms no sintético; **371–472 ms** num cofre real de 4.165 notas | ❌ |
| `note_read` p95 | ≤ 15 ms | **345 µs** | ✅ |
| `note_list` com filtro p95 | ≤ 10 ms | **534 µs** | ✅ |
| `vault_search` p95 | ≤ 100 ms | 7 de 8 formatos; `limit: 200` em 119–123 ms | ⚠️ |
| Reindexação de arquivo único | ≤ 20 ms | **335 µs** | ✅ |
| RSS em repouso | ≤ 60 MB | **38 MB** com cache quente, 55 MB a frio | ✅ |
| Processos órfãos após morte do host | 0 | **0 em 300 ciclos** | ✅ |

> [!WARNING]
> **Dois requisitos não estão totalmente atingidos, e estão medidos.** O boot com cache válido leva 371–472 ms num cofre real de 4.165 notas contra um teto de 300 ms — era 1192–1396 ms antes do cache de índice de metadados da Task 85 (2026-08-06), e o que sobra agora é a varredura de `Stat` por arquivo que confere o cache antes de aceitá-lo, mais cara num cofre sincronizado por nuvem que num disco local. `vault_search` atinge o teto em 7 dos 8 formatos medidos; só `limit: 200` continua acima, em 119–123 ms a 5.000 notas (era 181 ms). A reindexação de arquivo único e o RSS em repouso, que fechavam o M6 como não atingidos, passaram a ser atingidos com as otimizações de busca do M7 (Tasks 78–86) — ver a tabela de fechamento em [`docs/OPERACAO.md`](docs/OPERACAO.md). CPU em repouso e linearidade até 20.000 notas **não foram medidos**.
>
> A tabela dos 22 requisitos não-funcionais, cada um com número medido ou a palavra "não medido", está em [`docs/OPERACAO.md`](docs/OPERACAO.md).

O CI compara seis benchmarks contra uma referência versionada em `docs/bench-baseline.json` e **reprova o build** em regressão acima de 20%.

---

## 🛠️ Desenvolvimento

```bash
pwsh -File scripts/verify.ps1        # bateria inteira, para no primeiro erro
pwsh -File scripts/build.ps1         # binário com versão embutida
```

<details>
<summary>🔬 <b>O que a bateria cobre, e por que ela é um comando só</b></summary>

<br>

`verify.ps1` roda, em ordem e parando no primeiro erro: build, `go test -race`, os tetos de latência **sem** `-race`, `go vet` nos três alvos, `gofmt`, `golangci-lint`, a checagem de rede e a de parâmetros de schema.

Existe porque uma lista solta de comandos convida a rodar três dos cinco.

> [!IMPORTANT]
> `golangci-lint` precisa ser **v2.12.2**. O `go.mod` declara `go 1.25.0`, e um binário compilado com Go mais antigo recusa o config antes de analisar uma linha — um zero local não diz nada sobre o CI. Confira com `golangci-lint version`.

</details>

<details>
<summary>👻 <b>O gate de processos órfãos</b> — três cenários, um por mecanismo</summary>

<br>

```bash
# Os tres cenarios, que e o padrao
pwsh -File scripts/test_orphans.ps1 -Cycles 100

# Ou um de cada vez, para diagnosticar
pwsh -File scripts/test_orphans.ps1 -Cycles 100 -Scenario parent-death
```

| Cenário | O que morre | stdin do servidor | Único mecanismo possível |
|---|---|---|---|
| `stdin-eof` | o host | pipe cuja ponta de escrita o host segura | EOF |
| `parent-death` | o host intermediário | pipe segurado por um processo que **sobrevive** | vigília do pai |
| `signal` | **nada** | `CONIN$`, que nunca chega a EOF | `CTRL_BREAK` |

**Cada cenário desconecta os outros dois**, e o harness reprova se o motivo registrado não for o do mecanismo que o cenário nomeia. Sem isso, um cenário caindo no mecanismo errado pareceria verde — foi assim que a vigília do pai atravessou todos os marcos sem nunca ter sido exercitada ponta a ponta.

</details>

---

## 📚 Documentação

| Documento | Conteúdo |
|---|---|
| [`docs/PRD.md`](docs/PRD.md) | Problema, requisitos funcionais e não-funcionais, decisões fechadas, riscos, marcos |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Camadas, fluxos, modelo de dados, concorrência, ciclo de vida |
| [`docs/ESTRUTURA.md`](docs/ESTRUTURA.md) | Árvore de diretórios, responsabilidade de cada pacote, convenções |
| [`docs/TOOLS.md`](docs/TOOLS.md) | Contrato de cada tool: schemas de entrada e saída, códigos de erro |
| [`docs/WINDOWS.md`](docs/WINDOWS.md) | OneDrive, MAX_PATH, caixa de caminhos, fsnotify, registro no Claude Desktop |
| [`docs/OPERACAO.md`](docs/OPERACAO.md) | Medições, tabela completa de RNFs, diagnóstico e limites conhecidos |
