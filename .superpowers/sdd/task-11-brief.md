### Task 11: Teste de órfãos — critério de bloqueio de release

**Files:**
- Create: `scripts/test_orphans.ps1`
- Create: `scripts/build.ps1`
- Modify: `.github/workflows/ci.yml` (adiciona o job `orphans`, só Windows)

**Interfaces:**
- Consumes: `bin/gobsidian.exe` produzido por `scripts/build.ps1`
- Produces: script que sai com código 1 se qualquer `gobsidian.exe` sobreviver a 100 ciclos

- [ ] **Step 1: Escrever `scripts/build.ps1`**

Copie integralmente o script de `docs/WINDOWS.md` §7. Ele já está correto: `Set-StrictMode`, `CGO_ENABLED=0`, `-ldflags` com versão, commit e data, saída em ASCII.

- [ ] **Step 2: Escrever `scripts/test_orphans.ps1`**

`param()` precisa ser a **primeira instrução do arquivo** — antes de `Set-StrictMode`, antes de qualquer coisa. Fora dessa posição, o PowerShell trata o bloco como chamada de função e o script falha na primeira linha.

```powershell
param(
    [int]$Cycles = 100,
    [string]$VaultPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$BinaryPath = Join-Path $ProjectRoot "bin\gobsidian.exe"

if (-not (Test-Path $BinaryPath)) {
    Write-Warning "[!] Binario nao encontrado: $BinaryPath"
    exit 1
}

if (-not $VaultPath) {
    $VaultPath = Join-Path $ProjectRoot "testdata\vault_small"
}

Write-Output "[...] $Cycles ciclos de encerramento abrupto"

$Survivors = 0

for ($i = 1; $i -le $Cycles; $i++) {
    # O "host" e um cmd.exe que lanca o servidor e mantem o stdin dele aberto.
    # Matar o cmd com /F e o cenario que os tres mecanismos existem para cobrir.
    #
    # NAO use $Host como nome de variavel: e automatica do PowerShell e
    # somente-leitura, e a atribuicao falha.
    $HostProc = Start-Process -FilePath "cmd.exe" `
        -ArgumentList "/c", "`"$BinaryPath`" serve --vault `"$VaultPath`"" `
        -PassThru -WindowStyle Hidden

    Start-Sleep -Milliseconds 700

    $Children = Get-CimInstance Win32_Process -Filter "ParentProcessId = $($HostProc.Id)"
    $ServerPids = @($Children | Where-Object { $_.Name -eq "gobsidian.exe" } | ForEach-Object { $_.ProcessId })

    taskkill /F /PID $HostProc.Id 2>$null | Out-Null

    Start-Sleep -Seconds 2

    foreach ($ServerPid in $ServerPids) {
        $Alive = Get-Process -Id $ServerPid -ErrorAction SilentlyContinue
        if ($Alive) {
            $Survivors++
            Write-Warning "[!] Ciclo ${i}: PID $ServerPid sobreviveu"
            Stop-Process -Id $ServerPid -Force -ErrorAction SilentlyContinue
        }
    }

    if ($i % 10 -eq 0) { Write-Output "[i] $i/$Cycles" }
}

$Remaining = @(Get-Process -Name "gobsidian" -ErrorAction SilentlyContinue)
if ($Remaining.Count -gt 0) {
    Write-Warning "[!] $($Remaining.Count) processo(s) gobsidian.exe ainda em execucao"
    $Remaining | ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
    $Survivors += $Remaining.Count
}

if ($Survivors -gt 0) {
    Write-Warning "[!] FALHA: $Survivors orfao(s) em $Cycles ciclos"
    exit 1
}

Write-Output "[OK] Nenhum orfao em $Cycles ciclos"
exit 0
```

- [ ] **Step 3: Rodar localmente com poucos ciclos**

```powershell
.\scripts\build.ps1
.\scripts\test_orphans.ps1 -Cycles 10
```

Esperado: `[OK] Nenhum orfao em 10 ciclos`.

Se falhar, **não prossiga para M1.** O ciclo de vida é o requisito que define o produto; um servidor completo que deixa órfãos é pior que um servidor mínimo que não deixa. Diagnostique com `--log-level debug`: o log registra qual mecanismo disparou (`Reason()`), e um encerramento que nunca dispara nenhum aponta o mecanismo quebrado.

- [ ] **Step 4: Adicionar o job ao CI**

Em `.github/workflows/ci.yml`:

```yaml
  orphans:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: pwsh -NoProfile -File scripts/build.ps1
      - name: 100 ciclos de encerramento abrupto (RNF-10)
        run: pwsh -NoProfile -File scripts/test_orphans.ps1 -Cycles 100
```

- [ ] **Step 5: Rodar o ciclo completo**

Run: `.\scripts\test_orphans.ps1 -Cycles 100`
Esperado: `[OK] Nenhum orfao em 100 ciclos`. RNF-10 satisfeito.

- [ ] **Step 6: Commit e marcar M0**

```bash
git add scripts .github
git commit -m "test(lifecycle): 100-cycle orphan check as release gate"
git tag m0-lifecycle
```

**Fim de M0.** O produto responde a `initialize`, lista uma tool, diagnostica o ambiente e encerra corretamente sob qualquer forma de morte do host. Nada de domínio foi construído ainda, e é assim que deve ser.

---

## M1 — Leitura (v0.1)

O parser vem antes do índice porque é folha do grafo de dependências, puro, e integralmente testável por golden file. Um erro aqui contamina tudo acima silenciosamente.

---

