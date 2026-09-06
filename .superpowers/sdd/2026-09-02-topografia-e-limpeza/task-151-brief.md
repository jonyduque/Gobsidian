### Task 151: Lixeira move o arquivo em vez de copiar; a cópia de fallback recusa placeholder de nuvem (1.3)

`DeleteNote` com `to_trash` faz `ReadFile` + `WriteAtomic` + `Remove`
(`write.go:748-770`) — três operações e um arquivo inteiro na memória para o
que `os.Rename` faz numa. `moverCorpo` (`:805-841`) já é a conta de "mover
uma nota conferindo todos os erros"; a lixeira não a usa. E `moverCorpo`, no
fallback de cópia, faz `os.ReadFile` sem consultar `CloudOnly` — abrir um
placeholder do OneDrive dispara download síncrono, que é a regra "arquivo
somente-nuvem nunca é aberto" do `CLAUDE.md`, e "quem roda antes do guarda
precisa do mesmo guarda".

`PathLocker` NÃO é reentrante (`writer/lock.go:37-49`: `entry.mu.Lock()` de
novo na mesma goroutine trava para sempre). `DeleteNote` trava `canonical` em
`:713` e `moverCorpo` trava `de` e `para`; portanto a lixeira precisa chamar
`moverCorpo` SEM a trava de `DeleteNote` — a trava de `moverCorpo` cobre os
dois caminhos.

**Files:**
- Modify: `internal/service/write.go:700-777` (`DeleteNote`, ramo `ToTrash`) e `:826-841` (`moverCorpo`, fallback)
- Test: `internal/service/delete_test.go` (acrescentar teste portátil)
- Test: `internal/service/erro_engolido_windows_test.go` (acrescentar teste Windows)

- [ ] **Step 1: Teste portátil — a lixeira é um rename, não uma cópia**

Como saber que foi rename e não cópia+remove sem olhar a implementação? Um
arquivo grande em cópia custa tempo e memória, mas isso não é asserção. O que é
observável: rename preserva o mtime do arquivo; `WriteAtomic` (escreve, `sync`,
`rename` do temporário) produz um arquivo com mtime NOVO. Fixe um mtime antigo,
mova para a lixeira, confira que o mtime sobreviveu.

Em `internal/service/delete_test.go`:

```go
func TestDeleteNoteToTrashMoveSemCopiar(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "# A\n\ncorpo\n")
	antigo := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(root, "a.md"), antigo, antigo); err != nil {
		t.Fatal(err)
	}
	svc := newTestService(t, root)

	res, err := svc.DeleteNote(context.Background(), DeleteNoteRequest{Path: "a.md", ToTrash: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.TrashPath == "" {
		t.Fatalf("sem TrashPath: %+v", res)
	}
	fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(res.TrashPath)))
	if err != nil {
		t.Fatalf("a nota nao esta na lixeira: %v", err)
	}
	if !fi.ModTime().Equal(antigo) {
		t.Fatalf("mtime da copia na lixeira = %v, quer %v: a lixeira COPIOU em vez de mover", fi.ModTime(), antigo)
	}
	if _, err := os.Stat(filepath.Join(root, "a.md")); !os.IsNotExist(err) {
		t.Fatalf("a origem ainda existe (err=%v)", err)
	}
}
```

Acrescente `"os"`, `"path/filepath"`, `"time"` aos imports se faltarem.

- [ ] **Step 2: Rodar; DEVE falhar com "a lixeira COPIOU em vez de mover". Se passar sem o fix, a prova está errada (o `WriteAtomic` preservou mtime?) — pare, verifique com `os.Stat` antes/depois num teste descartável, e troque a asserção por outra que distinga rename de cópia (ex.: `os.SameFile` num handle aberto antes do move, no Windows). Não avance com um teste que já passa.**

- [ ] **Step 3: Teste Windows — a lixeira não abre placeholder**

Em `internal/service/erro_engolido_windows_test.go` (já é `//go:build windows`, pacote `service_test`, e tem `travaExclusiva`):

```go
// TestDeleteNoteToTrashNaoBaixaPlaceholder: com o rename recusado, o fallback
// de copia de moverCorpo faria os.ReadFile num placeholder de nuvem — o
// download sincrono que a regra "somente-nuvem nunca e aberto" proibe.
func TestDeleteNoteToTrashNaoBaixaPlaceholder(t *testing.T) {
	root := t.TempDir()
	caminho := filepath.Join(root, "nuvem.md")
	if err := os.WriteFile(caminho, []byte("# Nuvem\n\ncorpo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := windows.UTF16PtrFromString(vault.LongPath(caminho))
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
	}
	t.Cleanup(func() { _ = windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_NORMAL) })

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	if n, ok := idx.Get("nuvem.md"); !ok || !n.CloudOnly {
		t.Fatal("a nota nao ficou CloudOnly; o atributo nao pegou")
	}
	// Handle exclusivo: os.Rename recusa, e o fallback de copia e forcado.
	travaExclusiva(t, caminho)
	svc := service.New(v, idx, nil, nil, service.Options{})

	_, err = svc.DeleteNote(context.Background(), service.DeleteNoteRequest{Path: "nuvem.md", ToTrash: true})
	if err == nil {
		t.Fatal("DeleteNote to_trash devolveu sucesso sobre um placeholder com rename recusado: o fallback leu o arquivo")
	}
	if got := service.CodeOf(err); got != service.CodeCloudOnlyFile {
		t.Fatalf("codigo = %s, quer %s: %v", got, service.CodeCloudOnlyFile, err)
	}
}
```

Acrescente `"github.com/jonyd/gobsidian/internal/index"` e `".../internal/vault"` aos imports do arquivo.

- [ ] **Step 4: Rodar; falha (código diferente de CLOUD_ONLY_FILE, ou sucesso)**

- [ ] **Step 5: Reescrever o ramo `ToTrash` de `DeleteNote`**

Substitua de `unlock := s.locker.Lock(canonical)` (`:713`) até o fim do ramo `if req.ToTrash { ... }` (`:777`) por:

```go
	absPath := s.vault.Abs(canonical)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return DeleteNoteResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada no disco", req.Path)
	}

	if req.ToTrash {
		// Sem s.locker.Lock aqui: PathLocker nao e reentrante, e moverCorpo
		// trava origem E destino em ordem global. A lixeira e um move como
		// qualquer outro — ate 2026-09-02 era ReadFile + WriteAtomic +
		// Remove, tres operacoes e o arquivo inteiro em memoria para o que
		// os.Rename faz numa, e sem o guarda de placeholder que moverCorpo
		// ganhou junto com esta mudanca.
		trashRel, absTrash, err := s.destinoNaLixeira(canonical)
		if err != nil {
			return DeleteNoteResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(absTrash), 0755); err != nil {
			return DeleteNoteResult{}, Errorf(CodeInternal, "criando diretorio lixeira: %v", err)
		}
		if err := s.moverCorpo(ctx, canonical, vault.CanonicalPath(trashRel), absTrash); err != nil {
			return DeleteNoteResult{}, err
		}
		return DeleteNoteResult{
			Path:          string(canonical),
			Deleted:       true,
			MovedToTrash:  true,
			TrashPath:     trashRel,
			BrokenLinks:   brokenLinks,
			BrokenAnchors: brokenAnchors,
		}, nil
	}

	unlock := s.locker.Lock(canonical)
	defer unlock()
```

(O `os.Remove` da exclusão definitiva continua logo abaixo, agora sob a trava.)
E extraia a resolução do destino — é o bloco de `:721-741`, sem mudança de lógica:

```go
// destinoNaLixeira resolve .trash/<nome>, com sufixo de timestamp se ja
// houver um arquivo com esse nome la.
func (s *Service) destinoNaLixeira(canonical vault.CanonicalPath) (trashRel, absTrash string, err error) {
	baseName := filepath.Base(string(canonical))
	trashRel = filepath.ToSlash(filepath.Join(".trash", baseName))
	absTrash, _, err = vault.Resolve(s.vault.Root(), trashRel)
	if err != nil {
		return "", "", mapVaultErr(err)
	}
	if _, err := os.Stat(absTrash); err == nil {
		ext := filepath.Ext(baseName)
		stem := strings.TrimSuffix(baseName, ext)
		uniqueName := fmt.Sprintf("%s_%d%s", stem, time.Now().UnixNano(), ext)
		trashRel = filepath.ToSlash(filepath.Join(".trash", uniqueName))
		absTrash, _, err = vault.Resolve(s.vault.Root(), trashRel)
		if err != nil {
			return "", "", mapVaultErr(err)
		}
	}
	return trashRel, absTrash, nil
}
```

O comentário de `:757-765` ("O erro do remove e CONFERIDO...") migra para cima de `moverCorpo`, que já diz o mesmo — apague a cópia.

- [ ] **Step 6: Guarda de placeholder no fallback de `moverCorpo`**

Entre `os.Rename` e `os.ReadFile` (`:826-830`):

```go
	// Volume diferente, ou rename recusado: copia e remove, com o erro do
	// remove CONFERIDO. A copia LE a origem — e ler um placeholder de nuvem
	// dispara download sincrono. O rename de um placeholder nao baixa nada;
	// a copia baixa. Quem roda antes do guarda precisa do mesmo guarda.
	if n, ok := s.index.Get(de); ok && n.CloudOnly {
		return Errorf(CodeCloudOnlyFile,
			"nota %q e somente-nuvem e o rename foi recusado; a copia de fallback a baixaria", de)
	}
	fromRaw, err := os.ReadFile(absFrom)
```

- [ ] **Step 7: Rodar `go test ./internal/service -run 'Delete|Trash|Move' -race -v`**

Expected: os dois novos passam. `TestDeleteToTrashNaoMenteQuandoORemoveFalha`
(`erro_engolido_windows_test.go:41`) vai FALHAR na asserção
`strings.Contains(err.Error(), "lixeira")`: o erro agora vem de `moverCorpo`,
que diz "a nota foi copiada para %q mas a origem %q nao pode ser removida" com
o caminho `.trash/origem.md`. A mensagem não se adapta ao teste; o teste passa
a procurar `.trash` em vez de "lixeira" — é a mesma garantia (o erro diz onde
a cópia está) com a palavra que a conta única usa. Registre a troca no commit.

- [ ] **Step 8: Prova de mutação**

Apague o guarda do Step 6, rode `-run NaoBaixaPlaceholder`, cole a saída (deve
falhar com sucesso indevido ou código INTERNAL), restaure. O portátil já
provou no Step 2 que falhava com a cópia.

- [ ] **Step 9: Gate e commit**

```bash
pwsh -File scripts/verify.ps1
git add internal/service/write.go internal/service/delete_test.go internal/service/erro_engolido_windows_test.go
git commit -m "fix(service): trash moves the note instead of copying it, and the copy fallback refuses cloud placeholders"
```

#### Verificações
Além dos passos:
1. `TestDeleteNoteToTrashMoveSemCopiar` REPROVA antes do fix (mtime muda porque o arquivo foi copiado). Se ele passar antes do fix, o teste não mede o que promete — pare e reporte `BLOCKED` com a saída, não troque a asserção.
2. `PathLocker` não é reentrante: o caminho da lixeira NÃO passa pelo `Lock` do `DeleteNote` e depois pelo de `moverCorpo`. Um deadlock aqui aparece como teste que trava; o `go test` tem `-timeout` padrão de 10 min — use `-timeout 60s` nesta Task.
3. A guarda de `CloudOnly` está no fallback de cópia de `moverCorpo`, e o teste Windows a exercita com o rename FORÇADO a falhar (trava exclusiva). Sem forçar, o rename tem sucesso e o fallback não roda — o teste mediria o caminho principal.
4. `TestDeleteToTrashNaoMenteQuandoORemoveFalha` continua passando com a asserção ajustada para `.trash`.
5. `docs/TOOLS.md` em `note_delete`: `to_trash` move (rename) e descreve o fallback e o erro `CLOUD_ONLY_FILE`.

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde antes do commit, com a contagem de etapas colada no relatório. `-SkipCross -SkipNet` só para iterar.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`. Há trabalho não commitado no repositório (`test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`): `git diff <caminho>` antes de `git add <caminho>`; nunca `git add -A`.
- Nunca `go mod tidy`.
- Nunca despache subagentes. Nunca mate processo por nome (`Stop-Process -Name`); só por PID que você lançou.
- Referências e renames por gopls (LSP), não por grep. `grep` só para confirmar tags e reflection.
- Se um teste falhar por motivo que este brief não explica, **pare e reporte `BLOCKED`**; não ajuste a expectativa para o código passar.
- Este commit é de UMA categoria (o prefixo do assunto diz qual). Se você se pegar corrigindo outra coisa no caminho, anote em "O que ficou de fora" e não corrija.

#### Comando de mutação
Copie a âncora **do arquivo**, não de memória — âncora digitada sai `EXIT=2`. `0` = o teste reprovou sob mutação (o que se quer); `1` = a regra está escrita e não verificada; `2` = âncora ambígua ou build quebrado.

```bash
pwsh -File scripts/mutate.ps1 -Path internal/service/write.go `
  -Anchor 'if n, ok := s.index.Get(de); ok && n.CloudOnly {' `
  -Replacement 'if false {' `
  -Test TestDeleteNoteToTrashNaoBaixaPlaceholder -Package ./internal/service/
```
E, para a regra do move: mutação manual — troque a chamada a `moverCorpo` no ramo `ToTrash` por uma cópia (`os.ReadFile` + `WriteAtomic`), rode `TestDeleteNoteToTrashMoveSemCopiar`, cole a falha, restaure.

#### Contrato de relatório
Escreva o relatório completo no arquivo de relatório indicado no despacho; devolva no chat só status, SHA, uma linha de testes e as preocupações. O relatório traz:
- **Status** — `DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`.
- **Commit** — SHA curto e assunto (`git log -1 --oneline`).
- **Evidência de TDD** — comando do RED com a saída falhando; comando do GREEN com a saída passando. Não "segui TDD".
- **Prova de mutação** — para cada regra reivindicada: o comando `mutate.ps1` (ou a mutação manual, com o diff), qual teste reprovou **pelo nome e pela linha**, a saída colada, e a confirmação de restauro (`git diff --stat` limpo no arquivo mutado).
- **As verificações do brief** — cada uma com o resultado real, inclusive as que deram certo.
- **`verify.ps1`** — a última linha, com a contagem de etapas.
- **O que ficou de fora** — e por quê. Vazio é resposta aceitável; ausente não é.
- **`git status --porcelain`** — colado. Nenhum arquivo do usuário tocado.

---

