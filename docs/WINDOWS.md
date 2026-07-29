# Windows — gobsidian

Particularidades da plataforma que afetam o desenho, e não apenas a implementação. Este documento existe porque cofres reais no Windows vivem em pastas do OneDrive, com caminhos longos e casing inconsistente, e nenhuma dessas condições é tratada pelos servidores MCP existentes.

---

## 1. OneDrive

### 1.1 Arquivos somente-nuvem

Com *Files On-Demand* ativo, um arquivo pode existir como *placeholder*: a entrada de diretório está lá, com nome e tamanho corretos, mas o conteúdo não está no disco. Abrir o arquivo dispara download síncrono, que pode levar segundos ou falhar sem conexão.

Uma indexação ingênua abre todos os arquivos e, num cofre grande com muitos placeholders, força o download do cofre inteiro no boot — travando por minutos.

**Detecção.** Verificar os atributos do arquivo antes de abrir:

| Atributo | Significado |
|---|---|
| `FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS` (`0x00400000`) | Placeholder; ler dispara download |
| `FILE_ATTRIBUTE_RECALL_ON_OPEN` (`0x00040000`) | Abrir dispara download |
| `FILE_ATTRIBUTE_OFFLINE` (`0x00001000`) | Conteúdo não local |

Obtidos via `syscall.GetFileAttributes` (`golang.org/x/sys/windows`), sem abrir o arquivo.

**Comportamento.** Arquivos somente-nuvem são indexados apenas pelos metadados de diretório — caminho, tamanho, mtime — e marcados com `CloudOnly: true`. Não entram no índice de texto e não aparecem em `vault_search`. Aparecem em `note_list` e são contabilizados em `vault_stats`.

Ler explicitamente uma nota somente-nuvem via `note_read` **hidrata** o arquivo: é uma ação deliberada do usuário sobre um arquivo específico, com custo aceitável. Indexar o cofre inteiro não é.

Uma chamada que precise do conteúdo e encontre um placeholder inacessível falha com `CLOUD_ONLY_FILE` e uma mensagem que explica como resolver.

**Diagnóstico.**

```powershell
$VaultPath = "C:\Users\jonyd\OneDrive - Minha Organizacao\Meu Cofre\Meu Cofre"

$Files = Get-ChildItem -Path $VaultPath -Filter "*.md" -Recurse -File
$CloudOnly = $Files | Where-Object { $_.Attributes -band [System.IO.FileAttributes]::Offline }

$TotalCount = $Files.Count
$CloudCount = 0
if ($CloudOnly) { $CloudCount = @($CloudOnly).Count }

Write-Output "[i] Total de notas: $TotalCount"
Write-Output "[i] Somente nuvem:  $CloudCount"

if ($CloudCount -gt 0) {
    Write-Output "[*] Para hidratar o cofre inteiro:"
    Write-Output "    attrib -U +P `"$VaultPath\*`" /s"
}
```

### 1.2 Rajadas de eventos

O sincronizador do OneDrive gera eventos de sistema de arquivos que não correspondem a mudanças de conteúdo: atualização de metadados, transições de estado de sincronização, escrita de arquivos próprios de controle.

Sem filtragem, cada rajada dispara reindexação em massa. Esse é o motivo mais provável de um servidor MCP de Obsidian "ficar lento sem razão aparente" em máquinas com OneDrive.

**Mitigação em três camadas** (ver ARCHITECTURE §5.3):

1. Filtro de relevância: apenas `.md`, apenas fora dos diretórios excluídos
2. Debounce de 250 ms com coalescência por caminho
3. Verificação de mudança real: se mtime e tamanho batem com o índice, descartar sem parsear

A terceira camada é a que elimina o grosso do ruído do OneDrive, porque a maioria dos eventos espúrios não altera nenhum dos dois.

**Instrumentação.** `vault_stats` com `include_runtime` expõe eventos recebidos, coalescidos, processados e overflows. Razão alta entre recebidos e processados é o comportamento correto.

### 1.3 Violação de compartilhamento no rename

O OneDrive abre arquivos brevemente para calcular hash e enviar. Um `os.Rename` durante essa janela devolve `ERROR_SHARING_VIOLATION` (32).

**Mitigação.** Retry com backoff exponencial no `AtomicWrite`: três tentativas, 50 ms iniciais, dobrando. Falha após esgotar produz `FILE_LOCKED`. Na prática, a primeira retentativa resolve quase sempre.

### 1.4 Exclusões obrigatórias

Além dos diretórios padrão (`.obsidian/`, `.git/`, `.trash/`), o cofre em OneDrive precisa excluir os arquivos de controle do sincronizador. O padrão de exclusão inclui `~$*`, `*.tmp`, `.~lock.*` e `desktop.ini`.

---

## 2. Comprimento de caminho

O limite histórico de 260 caracteres (`MAX_PATH`) ainda se aplica por padrão. Um cofre com caminho base longo — pastas corporativas do OneDrive têm nomes extensos — consome boa parte do orçamento antes que qualquer nota exista.

**Contagem para o cofre de referência:**

```
C:\Users\jonyd\OneDrive - Minha Organizacao\Meu Cofre\Meu Cofre\
= 88 caracteres

Restam 172 para subpastas e nome de arquivo.
```

Suficiente na prática, mas não com folga confortável se a estrutura ganhar níveis.

**Mitigação no código.** Todo caminho absoluto passado a chamadas do sistema no Windows é prefixado com `\\?\` quando excede 240 caracteres. O prefixo eleva o limite para aproximadamente 32.767 caracteres.

Restrições do prefixo `\\?\`, que a camada `vault` precisa respeitar: exige caminho absoluto, exige separador `\` (não `/`), e não aceita `.` ou `..` — o caminho deve estar totalmente resolvido antes da prefixação.

**Mitigação no sistema.** Habilitar suporte a caminhos longos (Windows 10 1607+, requer privilégio de administrador):

```powershell
$RegPath = "HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem"
$Current = Get-ItemProperty -Path $RegPath -Name "LongPathsEnabled" -ErrorAction SilentlyContinue

if ($Current -and $Current.LongPathsEnabled -eq 1) {
    Write-Output "[OK] Caminhos longos ja habilitados"
}
else {
    Write-Output "[*] Habilitando caminhos longos (requer reinicio)"
    Set-ItemProperty -Path $RegPath -Name "LongPathsEnabled" -Value 1 -Type DWord
}
```

`gobsidian doctor` reporta o comprimento do caminho mais longo do cofre e avisa acima de 240.

---

## 3. Casing

O NTFS preserva maiúsculas e minúsculas mas as ignora na comparação. Cofres reais acumulam inconsistência entre pastas — `PENAL` ao lado de `Civil` ao lado de `CIVIL` — e wikilinks escritos com grafia divergente da do disco.

**Consequência arquitetural.** O índice armazena a grafia exata do disco no caminho canônico e mantém um mapa auxiliar de caminho em minúsculas para caminho canônico. A resolução tenta exato primeiro, depois insensível.

Se a busca insensível encontrar mais de um candidato — possível quando o cofre é acessado de um sistema sensível a maiúsculas, ou quando arquivos vieram de um repositório Git —, a chamada falha com `AMBIGUOUS_PATH` listando os candidatos. Escolher arbitrariamente seria pior que falhar.

**Detecção de colisão.**

```powershell
$VaultPath = "C:\caminho\do\cofre"

$Groups = Get-ChildItem -Path $VaultPath -Recurse -File -Filter "*.md" |
    Group-Object { $_.FullName.ToLowerInvariant() } |
    Where-Object { $_.Count -gt 1 }

if ($Groups) {
    Write-Warning "[!] Colisoes de casing encontradas:"
    foreach ($Group in $Groups) {
        $Names = $Group.Group | ForEach-Object { $_.FullName }
        $Joined = $Names -join "`n    "
        Write-Output "    $Joined"
    }
}
else {
    Write-Output "[OK] Nenhuma colisao de casing"
}
```

---

## 4. fsnotify no Windows

### 4.1 Como funciona

`fsnotify` usa `ReadDirectoryChangesW`. Diferenças relevantes em relação ao Linux:

- Não é recursivo nativamente na versão v1.10.1 — exige adicionar cada subdiretório manualmente ao observador, igual ao `inotify`
- Não há limite de watches por usuário, ao contrário do `max_user_watches` do Linux
- O buffer de eventos é finito e pode transbordar sob rajada
- Renomeação chega como par não correlacionado de remoção e criação

### 4.2 Overflow

Sob rajada intensa — sincronização inicial do OneDrive, `git checkout` de branch grande, extração de arquivo compactado — o buffer transborda e `fsnotify` emite `ErrEventOverflow`. Eventos foram perdidos e não se sabe quais.

**A única resposta correta é uma varredura completa de reconciliação:** percorrer o cofre, comparar mtime e tamanho contra o índice, reparsear divergências, remover do índice as notas que sumiram.

Ignorar o overflow deixa o índice silenciosamente incorreto — o pior estado possível, porque o servidor continua respondendo com confiança a partir de dados errados.

`vault_stats` expõe o contador de overflows. Ocorrências recorrentes indicam que a janela de debounce precisa ser ampliada via `--debounce-ms`. O menor valor aceito é `1`: zero é recusado na carga da configuração, porque sem coalescência a correlação de rename — que exige uma remoção e uma criação na mesma janela — para de detectar qualquer renomeação.

### 4.3 Handles e desligamento

Um watcher não fechado mantém *handles* de diretório abertos, o que impede a remoção da pasta e interfere na sincronização do OneDrive. Fechar o watcher é parte não opcional da sequência de shutdown (ARCHITECTURE §7.4).

---

## 5. Ciclo de vida do processo no Windows

### 5.1 Sinais

O Windows não tem sinais POSIX. O runtime do Go mapeia `os.Interrupt` para `CTRL_C_EVENT` e `CTRL_BREAK_EVENT`, e `syscall.SIGTERM` é aceito na API mas não é entregue por `taskkill` sem `/F`.

Consequência: **tratamento de sinal sozinho é insuficiente no Windows.** É exatamente por isso que os outros dois mecanismos existem.

### 5.2 EOF em stdin

O mecanismo primário, e o mais confiável no Windows. Quando o processo pai termina — de qualquer forma, incluindo `taskkill /F` —, o sistema operacional fecha os *handles* que ele detinha, e a leitura do stdin do filho retorna `io.EOF`.

### 5.3 Vigília do PID pai

Rede de segurança. Duas armadilhas específicas do Windows:

**Reutilização de PID.** PIDs são reciclados agressivamente. Verificar apenas a existência do PID produz falso negativo quando o PID do pai morto é atribuído a um processo novo. A verificação precisa comparar também o *creation time* do processo, obtido via `GetProcessTimes`.

**Obtenção do PID pai.** `os.Getppid()` funciona no Windows, mas o Windows não mantém relação pai-filho persistente: se o pai morre, o filho não é reparentado, e o PID pai registrado passa a apontar para nada ou para um processo reciclado. Capturar PID **e** creation time do pai no startup resolve ambos os casos.

### 5.4 Verificação de órfãos

```powershell
$Orphans = Get-CimInstance Win32_Process -Filter "Name = 'gobsidian.exe'"

if (-not $Orphans) {
    Write-Output "[OK] Nenhum processo gobsidian.exe em execucao"
}
else {
    foreach ($Proc in $Orphans) {
        $ProcId = $Proc.ProcessId
        $ParentId = $Proc.ParentProcessId
        $Parent = Get-CimInstance Win32_Process -Filter "ProcessId = $ParentId" -ErrorAction SilentlyContinue

        if (-not $Parent) {
            Write-Warning "[!] ORFAO: PID $ProcId (pai $ParentId nao existe mais)"
        }
        else {
            $ParentName = $Parent.Name
            Write-Output "[OK] PID $ProcId (pai: $ParentName)"
        }
    }
}
```

Este script é a base de `scripts/test_orphans.ps1`, que executa cem ciclos de iniciar e matar o host e falha se qualquer `gobsidian.exe` sobreviver. RNF-10, critério de bloqueio de release.

---

## 6. Codificação e fim de linha

### 6.1 UTF-8 e BOM

Notas do Obsidian são UTF-8. Editores do Windows podem introduzir BOM (`EF BB BF`) no início do arquivo.

O parser detecta e ignora o BOM na leitura, e o preserva na escrita se estava presente. Não removê-lo silenciosamente evita diffs espúrios em cofres versionados; não ignorá-lo na leitura evita que o primeiro heading do arquivo deixe de ser reconhecido.

### 6.2 Fim de linha

Arquivos no Windows podem ter CRLF, LF, ou mistura dos dois.

O estilo predominante é detectado no parse e armazenado em `Note.EOL`. Toda escrita normaliza o conteúdo novo para o estilo do arquivo — nunca converte o arquivo inteiro.

Sem isso, cada escrita produz um diff de arquivo inteiro em cofres versionados por Git, e a operação de "adicionar um parágrafo" fica indistinguível de "reescrever a nota".

### 6.3 Console

Saída de `gobsidian` em modo CLI usa apenas ASCII. Emojis e caracteres Unicode em console PowerShell dependem da página de código ativa e produzem saída ilegível quando ela é CP-850 ou CP-1252.

Marcadores de status: `[OK]`, `[!]`, `[*]`, `[i]`, `[...]`.

---

## 7. Build

```powershell
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$OutputDir = Join-Path $ProjectRoot "bin"

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

$Version = git describe --tags --always --dirty 2>$null
if (-not $Version) { $Version = "dev" }

$Commit = git rev-parse --short HEAD 2>$null
if (-not $Commit) { $Commit = "unknown" }

$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

$LdFlags = "-s -w -X main.version=$Version -X main.commit=$Commit -X main.buildDate=$BuildDate"
$OutputPath = Join-Path $OutputDir "gobsidian.exe"

Write-Output "[...] Compilando $Version ($Commit)"

$env:CGO_ENABLED = "0"
go build -ldflags $LdFlags -o $OutputPath ".\cmd\cofre"

if ($LASTEXITCODE -ne 0) {
    Write-Warning "[!] Falha na compilacao"
    exit 1
}

$SizeBytes = (Get-Item $OutputPath).Length
$SizeMB = [math]::Round($SizeBytes / 1MB, 2)
Write-Output "[OK] $OutputPath ($SizeMB MB)"
```

`CGO_ENABLED=0` garante binário estático sem dependência de DLL do sistema — parte do requisito de instalação trivial.

### Compilação cruzada

```powershell
$env:CGO_ENABLED = "0"

$env:GOOS = "windows"; $env:GOARCH = "amd64"; go build -o "bin\gobsidian-windows-amd64.exe" ".\cmd\cofre"
$env:GOOS = "darwin";  $env:GOARCH = "arm64"; go build -o "bin\gobsidian-darwin-arm64"     ".\cmd\cofre"
$env:GOOS = "linux";   $env:GOARCH = "amd64"; go build -o "bin\gobsidian-linux-amd64"      ".\cmd\cofre"

Remove-Item Env:\GOOS, Env:\GOARCH
```

---

## 8. Registro no Claude Desktop

### 8.1 Configuração

```powershell
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ConfigPath = Join-Path $env:APPDATA "Claude\claude_desktop_config.json"
$BinaryPath = Join-Path $env:USERPROFILE "go\bin\gobsidian.exe"
$VaultPath = "C:\Users\jonyd\OneDrive - Minha Organizacao\Meu Cofre\Meu Cofre"

if (-not (Test-Path $BinaryPath)) {
    Write-Warning "[!] Binario nao encontrado: $BinaryPath"
    exit 1
}

if (-not (Test-Path $VaultPath)) {
    Write-Warning "[!] Cofre nao encontrado: $VaultPath"
    exit 1
}

$Config = $null
if (Test-Path $ConfigPath) {
    $Config = Get-Content $ConfigPath -Raw | ConvertFrom-Json
}
else {
    $Config = [PSCustomObject]@{ mcpServers = [PSCustomObject]@{} }
}

if (-not $Config.PSObject.Properties.Name.Contains("mcpServers")) {
    $Config | Add-Member -MemberType NoteProperty -Name "mcpServers" -Value ([PSCustomObject]@{})
}

$Entry = [PSCustomObject]@{
    command = $BinaryPath
    args    = @("serve", "--vault", $VaultPath)
}

$Config.mcpServers | Add-Member -MemberType NoteProperty -Name "cofre" -Value $Entry -Force

$Config | ConvertTo-Json -Depth 10 | Out-File $ConfigPath -Encoding UTF8

Write-Output "[OK] gobsidian registrado em $ConfigPath"
Write-Output "[*] Reinicie o Claude Desktop para aplicar"
```

### 8.2 Armadilhas de configuração

**Aspas em excesso.** Caminhos com espaços não precisam de aspas adicionais quando são elementos separados do array `args`. Escrever `"--vault \"C:\\caminho com espaco\""` faz o servidor receber um caminho literalmente com aspas, que não existe. Este é o erro de configuração mais comum e o mais confuso de diagnosticar.

**Caminho relativo do binário.** O host MCP não herda necessariamente o `PATH` do shell do usuário. Use sempre caminho absoluto para `command`.

**Escape de barra invertida.** JSON exige `\\` para cada `\` literal. Um único `\` produz JSON inválido ou escape acidental.

**Barra invertida no fim do caminho.** Um caminho de cofre terminado em `\` vira `...\\"` no JSON e escapa a aspa de fechamento, quebrando o arquivo inteiro de forma que o erro reportado aponta para uma linha muito depois da real. Nunca termine o valor de `--vault` com separador.

**Encoding do arquivo.** `Out-File -Encoding UTF8` no Windows PowerShell 5.1 escreve BOM. O Claude Desktop tolera, mas para máxima compatibilidade prefira PowerShell 7+, onde `UTF8` significa sem BOM, ou use `-Encoding utf8NoBOM` explicitamente.

### 8.3 Diagnóstico

Quando o servidor não aparece no Claude Desktop, testar fora dele isola o problema:

```powershell
$BinaryPath = Join-Path $env:USERPROFILE "go\bin\gobsidian.exe"
$VaultPath = "C:\caminho\do\cofre"

# 1. Verificar o ambiente
& $BinaryPath doctor --vault $VaultPath

# 2. Handshake MCP manual
$InitRequest = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"manual","version":"1.0"}}}'

# O Start-Sleep nao e decorativo: mantem o stdin aberto tempo suficiente para
# a resposta ser escrita. Sem ele o pipe fecha no mesmo instante em que a
# requisicao entra, o servidor ve EOF, comeca a encerrar, e a resposta corre
# contra o encerramento — o resultado e stdout vazio e a conclusao errada de
# que o servidor esta quebrado. Um host MCP real mantem o stdin aberto pela
# sessao inteira, entao a condicao so aparece neste teste manual.
& {
    $InitRequest
    Start-Sleep -Seconds 2
} | & $BinaryPath serve --vault $VaultPath

# 3. Log de nivel debug para stderr
& $BinaryPath serve --vault $VaultPath --log-level debug 2> "gobsidian-debug.log"
```

O passo 2 deve produzir uma resposta JSON única em stdout. Se produzir qualquer outra coisa em stdout — banner, log, aviso —, essa é a causa da falha: **stdout pertence exclusivamente ao protocolo**.

---

## 9. Checklist do `gobsidian doctor`

| Verificação | Falha indica |
|---|---|
| Raiz do cofre existe e é diretório | Caminho errado na configuração |
| Permissão de leitura na raiz | Problema de ACL |
| Permissão de escrita (a menos que `--read-only`) | Cofre em pasta protegida |
| `.obsidian/` presente | Provavelmente não é um cofre do Obsidian |
| Contagem de notas | Cofre vazio ou caminho errado |
| Caminho mais longo do cofre | Acima de 240: risco de MAX_PATH |
| Suporte a caminhos longos no registro | Desabilitado com caminhos longos presentes |
| Arquivos somente-nuvem | Indexação incompleta esperada |
| Colisões de casing | Resolução ambígua de caminho |
| Espaço livre em disco | Escritas atômicas exigem espaço para o temporário |
| Diretório de cache acessível | Cache de índice desabilitado, boot mais lento |
| Versão do sincronizador de nuvem detectada | Contexto para diagnóstico de eventos |

Saída em ASCII, com `[OK]`, `[!]` e `[*]`. Código de saída zero quando não há falha bloqueante.
