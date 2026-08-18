# Relatorio da Task 105

- **Status**: DONE
- **Commit**: `c9d0da2` (ci: run orphan daemon-idle scenario and add checking scripts job)

## O que mudou
- `.github/workflows/ci.yml`: Adicionado o quarto cenario `daemon-idle` ao job `orphans` e criado o novo job `checagens` executando `check_doc_refs.ps1`, `check_readme_anchors.ps1` e `check_tool_params.ps1`.

## Evidencia de TDD (RED / GREEN)
- N/A: Tarefa de edicao de YAML do workflow do CI, sem desenvolvimento de codigo de producao Go.

## Prova de mutacao
- N/A: Tarefa de edicao de YAML do workflow do CI, sem alteracoes em codigo Go.

## Verificacoes locais dos scripts

### 1. `check_doc_refs.ps1` (sem pipe)
```
$ pwsh -File scripts/check_doc_refs.ps1
[i] corpus: 260 arquivos .go.
[i] 25 dispensa(s) em uso -- nao contam como achado:
[OK] nenhum token entre crases parece citar artefato ausente do codigo.
EXIT=0
```

### 2. `check_readme_anchors.ps1` (sem pipe)
```
$ pwsh -File scripts/check_readme_anchors.ps1
[i] 11 heading(s), 11 link(s) interno(s).
[OK] toda ancora resolve e toda secao H2 e alcancavel pela navegacao.
EXIT=0
```

### 3. `check_tool_params.ps1` (sem pipe)
```
$ pwsh -File scripts/check_tool_params.ps1
[!] 2 parametro(s) declarado(s) e nunca lido(s):
    tools_read.go:328  tagListInput.Sort  (json: "sort") -> nunca lido no handler da tool (mcpsrv)
    tools_read.go:329  tagListInput.Hierarchical  (json: "hierarchical") -> repassado ao dominio mas nunca lido em internal/
EXIT=1
```
(Nota: EXIT=1 e o comportamento esperado conforme Task 104 ate a Task 120 resolver/remover os parametros `tagListInput.Sort` e `tagListInput.Hierarchical`).

### 4. `test_orphans.ps1 -Cycles 20 -Scenario daemon-idle` (local)
```
$ pwsh -File scripts/test_orphans.ps1 -Cycles 20 -Scenario daemon-idle
[...] 20 ciclos de ociosidade do daemon (sem cliente conectado, sem pai vigiavel)
[OK] Nenhum daemon orfao em 20 ciclos -- todos sairam por ociosidade (reason=idle) apos a unica ponte ser morta
EXIT=0
```
(No CI o numero de ciclos e 100).

## Diff de `.github/workflows/ci.yml`
```diff
diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
index a6c1d42..846c1e9 100644
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -53,6,23 @@ jobs:
         shell: pwsh
         run: ./scripts/check_net.ps1
 
+  checagens:
+    runs-on: ubuntu-latest
+    steps:
+      - uses: actions/checkout@v4
+      - uses: actions/setup-go@v5
+        with:
+          go-version: '1.25'
+      - name: referencias a codigo em documentos
+        shell: pwsh
+        run: ./scripts/check_doc_refs.ps1
+      - name: ancoras do README resolvidas e secoes alcancaveis
+        shell: pwsh
+        run: ./scripts/check_readme_anchors.ps1
+      - name: parametros de tools declarados e lidos
+        shell: pwsh
+        run: ./scripts/check_tool_params.ps1
+
   lint:
     runs-on: ubuntu-latest
     steps:
@@ -131,3 +148,6 @@ jobs:
       - name: 100 ciclos - sinal, com stdin aberto e pai vivo
         shell: pwsh
         run: ./scripts/test_orphans.ps1 -Cycles 100 -Scenario signal
+      - name: 100 ciclos - daemon ocioso
+        shell: pwsh
+        run: ./scripts/test_orphans.ps1 -Cycles 100 -Scenario daemon-idle
```

## `git status --porcelain`
```
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
```

## O que ficou de fora
- N/A
