# Relatório: publicação da v1.0.0 e do instalador

**Data:** 2026-08-03
**Commits:** `9220aba` (correção do workflow de release), `581e5e7` (idempotência
do instalador), e os do instalador entre `a98e2e3` e `29c8f58`.
**Tag:** `v1.0.0` → `9220aba`. **Release publicada**, com quatro assets.

---

## O que existe agora, verificado

| Coisa | Estado | Conferido por |
|---|---|---|
| Repositório público | sim | `gh repo view` → `visibility=PUBLIC` |
| Tag `v1.0.0` no remoto | sim, em `9220aba` | `git ls-remote --tags` |
| Release `v1.0.0` | sim, não-draft, não-prerelease | `gh release view` |
| Assets | 4 — três binários e `SHA256SUMS.txt` | `gh release view --json assets` |
| Instalador de uma linha | funciona ponta a ponta | rodada real, abaixo |

---

## Evidência de TDD e prova de mutação

**Não há prova de mutação aqui, por construção**: o entregável é uma publicação e
um instalador, não uma regra de código com teste Go. `scripts/mutate.ps1` roda
teste Go com `-Test` e `-Package`, e nada disto é teste Go. O que a substitui é a
saída real de cada rodada, colada.

O ciclo existiu, e o vermelho foi encontrado rodando o instalador de verdade, não
imaginado:

### RED

Quatro falhas reais, cada uma numa rodada contra a release publicada:

```
[!] SHA256SUMS.txt nao foi encontrado nesta release. Instalacao abortada.
[!] Claude Code (CLI): claude mcp add saiu 1
    (claude dizia: unknown option '--vault')
[!] Claude Code (CLI): claude mcp add saiu 1: MCP server gobsidian already exists in user config
gobsidian v1.0.0 (6c092b4) 2026-58-08/03/26Thh:mm:ssZ
```

A terceira linha só pôde ser lida porque a segunda motivou parar de descartar a
saída dos CLIs — até então a falha aparecia como um código de saída nu.

### GREEN

Depois das quatro correções, contra a release real e sem fixar versão:

```
[OK] SHA-256 confere (2d785e15407c0a6f...)
[i] gobsidian v1.0.0 (9220aba) 2026-08-03T12:14:14Z
   ok        Claude Desktop, Claude Code (CLI), Gemini CLI
```

E duas rodadas seguidas, para a idempotência:

```
=== rodada 1 ===    ok        Claude Desktop, Claude Code (CLI), Gemini CLI
=== rodada 2 ===    ok        Claude Desktop, Claude Code (CLI), Gemini CLI
```

---

## Quatro defeitos que só apareceram com uma release de verdade

Nenhum deles era visível antes de existir algo publicado para consumir.

### 1. O workflow de release nunca publicou `SHA256SUMS.txt`

Nem a v0.1.0 nem a primeira v1.0.0 traziam o arquivo. O instalador **recusa
instalar** um binário cuja soma não foi publicada, e foi essa recusa que expôs a
lacuna:

```
[OK] Baixado: gobsidian-windows-amd64.exe
[!] SHA256SUMS.txt nao foi encontrado nesta release. Instalacao abortada.
```

O passo de geração entrou em `release.yml`.

### 2. A data de build era lixo, desde sempre

```yaml
BUILD_DATE=$(date -u +"%Y-%M-%DThh:mm:ssZ")
```

No `date(1)`, `%M` é o **minuto** e `%D` é a data no formato americano; `hh`, `mm`
e `ss` minúsculos não são especificadores e saem literais. Confirmado baixando o
binário publicado e rodando `version` nele:

```
gobsidian v1.0.0 (6c092b4) 2026-58-08/03/26Thh:mm:ssZ
```

Depois da correção, o binário da release atual responde:

```
gobsidian v1.0.0 (9220aba) 2026-08-03T12:14:14Z
```

### 3. O PowerShell come o separador `--`

O primeiro teste ponta a ponta não registrou nada no Claude Code:
`claude mcp add saiu 1`, e o próprio `claude` dizia `unknown option '--vault'`.

O PowerShell trata `--` como token de fim-de-parâmetros e o **remove** antes que o
comando nativo o veja — mas só quando ele aparece literalmente na linha. Dentro de
um array splatado, atravessa intacto. Medido nas duas formas contra o CLI
instalado: literal sai 1, splatado sai 0 e o servidor conecta.

### 4. Rodar o instalador duas vezes falhava

`claude mcp add` recusa um nome que já existe. O caminho de JSON deste mesmo
instalador atualiza no lugar, então os dois se comportavam de formas diferentes.

Esse motivo **só ficou visível** porque, no meio do caminho, o instalador parou de
jogar a saída dos CLIs no `Out-Null`: até então a falha aparecia como um código de
saída nu. Agora os três hosts de CLI removem antes de acrescentar.

Provado rodando duas vezes seguidas contra a release publicada:

```
=== rodada 1 ===    ok        Claude Desktop, Claude Code (CLI), Gemini CLI
=== rodada 2 ===    ok        Claude Desktop, Claude Code (CLI), Gemini CLI
```

---

## A prova ponta a ponta

Uma pré-release descartável (`v1.0.0-installer-test`) foi criada só para provar o
caminho feliz antes da release definitiva, e removida em seguida. A prova final é
contra a release real, **sem fixar versão**:

```
[OK] Versao: v1.0.0
[OK] Baixado: gobsidian-windows-amd64.exe
[OK] SHA-256 confere (2d785e15407c0a6f...)
[OK] Binario em ...\gobsidian.exe
[i] gobsidian v1.0.0 (9220aba) 2026-08-03T12:14:14Z
[OK] Cofre: ...
[OK] encontrado: Claude Desktop
[OK] encontrado: Claude Code (CLI)
[OK] encontrado: Gemini CLI
[OK] encontrado: Antigravity
[OK] encontrado: Antigravity IDE
[OK] encontrado: VS Code
   ok        Claude Desktop, Claude Code (CLI), Gemini CLI
```

O servidor foi conferido **conectado**, não apenas registrado:

```
gobsidian: ...\gobsidian.exe serve --vault ... - OK Connected
```

E a fusão de configuração preservou o que já existia na máquina:

```
servidores: ['gobsidian', 'obsidian2']
chaves de topo: ['coworkUserFilesPath', 'mcpServers', 'preferences']
```

Os seis hosts detectados são os que estão de fato instalados. Cursor, Windsurf e
Codex **não** foram oferecidos, e é o comportamento certo: existem `~/.cursor` e
`~/.codex` nesta máquina, criados por outra ferramenta, sem nenhum dos dois
produtos instalado. A detecção olha o executável ou a instalação, nunca o
diretório de configuração sozinho.

---

## Reescrita de histórico, e o que ela custou

Tornar o repositório público exigiu limpar dados pessoais que estavam em 15
arquivos e 12 commits: nome do empregador, caminho do cofre e nome da máquina.

- Backup em bundle antes de tocar em qualquer coisa, e conferido.
- `git filter-repo` sobre 289 commits. Zero ocorrências restantes, verificado com
  `git log --all -p`.
- **374 SHAs remapeados** em 86 arquivos a partir do `commit-map`, recusando
  qualquer prefixo curto que casasse com mais de um commit antigo. Nenhum era
  ambíguo.
- `audit_reports.ps1` voltou a ter só o falso positivo conhecido.

A conferência achou um defeito antigo de brinde: a **Task 28 estava registrada num
SHA que nunca existiu**. Confirmado contra o bundle anterior à reescrita, então
não veio dela. É a mesma classe do que já estava anotado na Task 31.

---

## O que ficou de fora

- **O caminho do Codex não foi verificado.** O `codex` não está instalado na
  máquina onde o instalador foi desenvolvido. Ele usa a mesma forma splatada que
  o `claude`, que está provada, mas isso é inferência e não medição. É por esse
  motivo que uma falha de host é reportada na própria linha e não derruba os
  demais.
- **Cursor e Windsurf idem**, pelo mesmo motivo: escrevem JSON no formato que os
  outros arquivos `mcp_config.json` desta máquina usam, mas nenhum dos dois
  produtos existe aqui para confirmar.
- **`gobsidian version` em Linux e macOS.** Só o binário de Windows foi
  executado; dos outros dois, conferiu-se o formato.
- **Os três RNFs não atingidos** seguem não atingidos e documentados em
  `docs/OPERACAO.md`. A publicação não mudou nenhum número.
