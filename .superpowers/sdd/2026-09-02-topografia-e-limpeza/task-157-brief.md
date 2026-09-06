### Task 157: A versão do cache de busca é uma conta só (1.12, fecha c2 por ora)

`search/persist.go:23` `CacheFormatVersion = 6`; `persist_codec.go:47-48`
`cacheMagic = "GBS6"`, `cacheCodecVers = 6`. Três literais para um número. O
próximo bump de formato que esqueça um deles produz um cache que passa pelo
cabeçalho e falha na decodificação — ou pior, decodifica lixo estruturalmente
válido (é o que o comentário de `:480-484` teme). O dono decidiu (c2) que
`internal/bincodec` fica adiado; o alias fecha o defeito agora.

**Files:**
- Modify: `internal/search/persist_codec.go:46-49`
- Test: `internal/search/persist_codec_test.go` (acrescentar, pacote `search`)

- [ ] **Step 1: Teste**

```go
func TestVersaoDoCacheDeBuscaEUmaConta(t *testing.T) {
	if cacheCodecVers != CacheFormatVersion {
		t.Fatalf("cacheCodecVers = %d, CacheFormatVersion = %d: duas contas", cacheCodecVers, CacheFormatVersion)
	}
	if quer := fmt.Sprintf("GBS%d", CacheFormatVersion); cacheMagic != quer {
		t.Fatalf("cacheMagic = %q, quer %q", cacheMagic, quer)
	}
}
```

- [ ] **Step 2: Rodar; passa (os três valem 6 hoje). Agora a prova: mude `CacheFormatVersion` para 7 em `persist.go`, rode, o teste FALHA nas duas asserções; restaure. Cole a saída.**

- [ ] **Step 3: Alias**

```go
const (
	// Uma conta: CacheFormatVersion e o numero; o magic e o codec derivam dele.
	// Ate 2026-09-02 eram tres literais, e um bump que esquecesse um deles
	// produziria um cache que passa pelo cabecalho e falha — ou decodifica
	// lixo estruturalmente valido — no corpo.
	cacheCodecVers = CacheFormatVersion
)

var cacheMagic = fmt.Sprintf("GBS%d", CacheFormatVersion)
```

`cacheMagic` deixa de ser `const`; os usos (`:178`, `:457-462`) já o tratam como string e compilam. Se `fmt` não estiver importado em `persist_codec.go`, acrescente. Se preferir manter `const` — `cacheMagic = "GBS" + string(rune('0'+CacheFormatVersion))` só vale até 9; o `var` é mais honesto.

- [ ] **Step 4: Repetir o Step 2 (bump para 7 → agora o teste PASSA, porque tudo derivou; e `TestLoadInvertedCache*` devem seguir passando com cache gravado e lido pela mesma versão). Restaure. Rodar `go test ./internal/search`.**

- [ ] **Step 5: Gate, commit**

```bash
git add internal/search/persist_codec.go internal/search/persist_codec_test.go
git commit -m "fix(search): the cache format version is one constant; magic and codec derive from it"
```

#### Verificações
Além dos passos:
1. A prova de mutação desta Task é o bump temporário (Step 2 e Step 4): ANTES do alias, bump para 7 faz o teste reprovar; DEPOIS, bump para 7 faz tudo derivar e o teste passa. Cole as DUAS saídas.
2. `grep -rn "GBS6\|cacheCodecVers = 6" internal/search` vazio depois.
3. Um cache gravado por esta versão e lido por ela mesma continua carregando: `go test ./internal/search -run 'Cache|Persist' -v` verde.

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

Não há `mutate.ps1` aqui: a prova é o bump manual de `CacheFormatVersion` para 7 descrito nos Steps 2 e 4, com as duas saídas coladas e o restauro confirmado por `git diff --stat internal/search/persist.go` vazio.

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

