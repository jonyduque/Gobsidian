### Task 156: A CLI `search` respeita `--max-results`; `index` e `inspect` param de aceitar flags que ignoram (1.11 + 5.8)

`cmd/gobsidian/search.go:50` monta `service.Options{ReadOnly: cfg.ReadOnly}` —
`cfg.MaxResults` é lido, validado e descartado; a flag `--max-results` do
`search` (`:97`) promete um teto que não aplica. `index.go:91-93` e
`inspect.go:129-131` declaram `--read-only`, `--debounce-ms` e `--max-results`
que nenhum dos dois subcomandos usa: nem escrevem, nem observam, nem buscam.
"Schema que promete e código que ignora é pior que parâmetro ausente".

**Files:**
- Modify: `cmd/gobsidian/search.go:26-28`, `:50`, `:93-94` (tirar `--read-only` e `--debounce-ms`; manter `--max-results`)
- Modify: `cmd/gobsidian/index.go:32-34`, `:91-93`; `cmd/gobsidian/inspect.go:37-39`, `:129-131` (tirar as três)
- Modify: `README.md:196-206` (tabela de flags: dizer quais subcomandos aceitam cada uma)
- Test: `cmd/gobsidian/cli_subcommands_test.go` (acrescentar)

- [ ] **Step 1: Teste**

Veja como `cli_subcommands_test.go` executa um subcomando (procure `newSearchCmd()` ou `SetArgs`); siga o mesmo padrão:

```go
func TestSearchCLIRespeitaMaxResults(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		escreve(t, root, fmt.Sprintf("n%d.md", i), "# Nota\n\npalavra unica aqui\n")
	}
	var out bytes.Buffer
	cmd := newSearchCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--vault", root, "--json", "--max-results", "2", "palavra"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var res struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("saida nao e JSON: %v\n%s", err, out.String())
	}
	if len(res.Results) != 2 {
		t.Fatalf("results = %d com --max-results 2: a flag e lida e descartada", len(res.Results))
	}
}

func TestIndexEInspectNaoAceitamFlagsQueIgnoram(t *testing.T) {
	for _, tc := range []struct {
		nome string
		cmd  func() *cobra.Command
	}{{"index", newIndexCmd}, {"inspect", newInspectCmd}} {
		for _, flag := range []string{"read-only", "debounce-ms", "max-results"} {
			if tc.cmd().Flags().Lookup(flag) != nil {
				t.Errorf("%s declara --%s e nao a usa", tc.nome, flag)
			}
		}
	}
	for _, flag := range []string{"read-only", "debounce-ms"} {
		if newSearchCmd().Flags().Lookup(flag) != nil {
			t.Errorf("search declara --%s e nao a usa", flag)
		}
	}
}
```

`escreve` é o helper de escrita de arquivo que o arquivo de teste já tem (confira o nome; se não houver, `os.WriteFile` direto). Confira os nomes reais dos construtores (`newIndexCmd`, `newInspectCmd`) com `grep -n "^func new.*Cmd" cmd/gobsidian/*.go`.

- [ ] **Step 2: Rodar; o primeiro falha com `results = 5`, o segundo lista seis flags**

- [ ] **Step 3: `search.go`**

`:50`: `service.New(v, idx, inv, nil, service.Options{ReadOnly: cfg.ReadOnly, MaxResults: cfg.MaxResults})` — confira o nome do campo em `service.Options` (`grep -n "MaxResults" internal/service/service.go`). Apague `:26-27` (`ReadOnlySet`, `DebounceMSSet`) e `:93-94` (as duas flags). `cfg.ReadOnly` segue vindo de `GOBSIDIAN_READ_ONLY` via `config.Load`; a CLI de busca nunca escreve, então `ReadOnly` em `Options` é indiferente — mantenha para não mudar o comportamento neste commit.

- [ ] **Step 4: `index.go` e `inspect.go`**: apague as três linhas `flags.*Set = ...` e as três `cmd.Flags()...` em cada um.

- [ ] **Step 5: `README.md:196-206`**: na tabela de flags, acrescente uma coluna "Subcommands" — `--vault`: all; `--read-only`, `--debounce-ms`, `--cache-dir`, `--eager-search`, `--log-level`: `serve`; `--max-results`: `serve`, `search`; `--follow-symlinks`: all; `--json`, `--limit`: `search` (e `inspect` se tiver `--json`). Confira cada flag contra o código antes de escrever a linha — a tabela é contrato.

Run: `pwsh -File scripts/check_readme_anchors.ps1`.

- [ ] **Step 6: Rodar os dois testes; gate; commit**

```bash
git add cmd/gobsidian/search.go cmd/gobsidian/index.go cmd/gobsidian/inspect.go cmd/gobsidian/cli_subcommands_test.go README.md
git commit -m "fix(cli): search honours --max-results; index and inspect stop declaring flags they ignore"
```

#### Verificações
Além dos passos:
1. `gobsidian search --help`, `index --help`, `inspect --help` — cole os três; `search` mostra `--max-results` e não mostra `--read-only`/`--debounce-ms`; os outros dois não mostram nenhuma das três.
2. `README.md` — a tabela de flags diz, por flag, quais subcomandos a aceitam; `pwsh -File scripts/check_readme_anchors.ps1` verde.
3. `cfg.MaxResults` chega a `service.Options` — o teste prova; e o `--limit` do `search` (se existir) continua funcionando como antes — cole um `search --json --limit 3`.

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
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/search.go `
  -Anchor 'MaxResults: cfg.MaxResults' `
  -Replacement 'MaxResults: 0' `
  -Test TestSearchCLIRespeitaMaxResults -Package ./cmd/gobsidian/
```

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

