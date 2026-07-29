### Task 25: Paridade com o Obsidian

#### CORRECAO OBRIGATORIA — o teste de paridade passa sem verificar nada (auditoria 2026-07-28)

O que entrou tem `testdata/parity/metadata.json` com tres bytes (`{}`) e `testdata/parity/vault/` **vazio**. O guard checa `os.Stat` do diretorio — que existe — entao **nao pula**. O laco `for path, want := range ref` nunca executa. Resultado: `PASS`.

Isso e pior que qualquer das duas alternativas. A metrica de sucesso mais forte do PRD (§7, divergencia zero contra o Obsidian) aparece como atingida sem nada ter sido comparado.

**O guard precisa checar conteudo, nao existencia.** Pule quando o corpus nao tiver nota nenhuma, ou quando a referencia estiver vazia:

```go
func TestParityWithObsidian(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "parity", "vault")
	refPath := filepath.Join("..", "..", "testdata", "parity", "metadata.json")

	// Checar CONTEUDO, nao existencia. Um diretorio vazio e um metadata.json
	// com "{}" fazem o laco de comparacao nao executar nenhuma vez, e o teste
	// passa afirmando uma paridade que ninguem verificou.
	notes, _ := filepath.Glob(filepath.Join(root, "*.md"))
	sub, _ := filepath.Glob(filepath.Join(root, "*", "*.md"))
	if len(notes)+len(sub) == 0 {
		t.Skip("corpus de paridade vazio; ver tools/parity-dumper/README.md")
	}

	ref := loadReference(t, refPath)
	if len(ref) == 0 {
		t.Skip("referencia de paridade vazia; rode o plugin dumper — ver tools/parity-dumper/README.md")
	}

	// ... resto inalterado
}
```

Prove que o skip acontece: rode com o estado atual e confirme `SKIP`, com a mensagem apontando o README. Depois, com um corpus de uma nota so e uma referencia de uma entrada, confirme que ele **roda** e compara.

#### Onde isto encaixa

Todo o parser e todo o índice estão prontos e testados contra golden files — mas golden file prova apenas que o comportamento não mudou, não que ele está **certo**. Esta tarefa compara o resultado contra o Obsidian real.

É a métrica que distingue um grafo correto de um grafo plausível, e a única que o próprio produto não consegue arbitrar sozinho.

#### Esta tarefa exige uma etapa humana e não pode ser totalmente automatizada

O *metadata cache* do Obsidian vive no IndexedDB do Electron. Não é arquivo que se leia de fora. A referência sai de um plugin de desenvolvimento descartável, rodado **uma vez**, por uma pessoa, num cofre de teste:

1. Compilar o plugin de `tools/parity-dumper/` e copiá-lo para `<cofre-de-teste>/.obsidian/plugins/parity-dumper/`
2. Habilitar o plugin no Obsidian
3. Rodar o comando que serializa `app.metadataCache`
4. Mover o `metadata.json` resultante para `testdata/parity/`

Se o corpus e o `metadata.json` **não** estiverem presentes, o teste deve pular com mensagem apontando para `tools/parity-dumper/README.md` — não falhar. Um teste que falha por falta de artefato humano é um teste que as pessoas aprendem a ignorar.

#### O que já está fechado e vincula esta tarefa

- **A comparação é assimétrica de propósito.** Nossa saída precisa conter tudo o que o Obsidian encontrou; o inverso não é exigido, porque âncoras quebradas e formas de resolução são informação que ele não expõe. Divergência para menos é falha; para mais, é o produto.
- **Divergência é bug do parser ou da resolução, não do teste.** Ajustar a expectativa para o teste passar é exatamente o que esta tarefa existe para impedir. Casos em que o Obsidian se comporta de forma que consideramos errada devem ser documentados com o motivo, em `tools/parity-dumper/README.md`, e só então excluídos.
- **Já há uma lista de perguntas de paridade** acumulada pelas revisões: delimitador de fechamento de frontmatter com espaço final, chave duplicada no YAML, `[[[triplo]]]`, `[[a\|b]]` com pipe escapado, tag com emoji. Confira cada uma contra a referência e registre a resposta.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Mensagem de falha inútil.** Uma falha que diz apenas "divergência em N.md" custa mais tempo do que economiza. Cada asserção precisa reportar o caminho, o que o Obsidian viu e o que nós vimos.
- **Corpus que não exercita nada.** As fixtures da Task 8 usavam extensões que o filtro descartaria de qualquer jeito. Aqui: um corpus só de notas simples não testa resolução. Inclua colisão de nome entre pastas, aliases colidentes, âncoras para headings renomeados, embeds de anexo e links para notas inexistentes.

#### Verificações além dos passos

- Sem `testdata/parity/`, o teste **pula** com mensagem acionável? Prove rodando com o diretório ausente.
- Com o corpus presente, cada `assert` reporta caminho, valor esperado e valor obtido?
- Cada pergunta de paridade acumulada tem resposta registrada?

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o código desta seção; não improvise uma variante. Se ele não compilar, corrija o erro mecânico e **diga exatamente o que mudou**. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar. Teste dobrado para passar é como defeito silencioso chega em produção.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** Há trabalho não commitado de outras frentes neste repositório, e um subagente já destruiu trabalho exatamente assim. Para desfazer o que você escreveu, edite de volta ou apague o arquivo específico que você criou.
- **`go mod tidy` está proibido.** Várias dependências fixadas ainda não têm importador, e o `tidy` as removeria — inclusive o pin obrigatório do SDK de MCP. Se o build reclamar de entrada faltando em `go.sum`, **pare e reporte**; não rode `go get`.
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.** Escrita em modo texto converte o arquivo inteiro para CRLF no Windows e o `gofmt` rejeita. Já custou dois commits neste projeto.
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês. Sem arquivos chamados `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave o relatório completo em `.superpowers/sdd/task-25-report.md`, com: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a tabela de verificações extras acima com o resultado real de cada uma; arquivos alterados; achados da auto-revisão; correções mecânicas que fez no código do plano; e preocupações.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, as respostas diretas pedidas acima, e preocupações. O detalhe mora no arquivo de relatório, não na resposta.

**Files:**
- Create: `tools/parity-dumper/manifest.json`, `tools/parity-dumper/main.ts`, `tools/parity-dumper/README.md`
- Create: `internal/index/parity_test.go`
- Create: `testdata/parity/vault/**`, `testdata/parity/metadata.json`

**Interfaces:**
- Consumes: `Index` (Tasks 19–22)
- Produces: teste que compara nosso grafo contra o `metadataCache` real do Obsidian

- [ ] **Step 1: Escrever o plugin dumper**

`tools/parity-dumper/main.ts`:

```ts
import { Plugin, TFile, Notice } from "obsidian";

// Plugin de desenvolvimento descartavel. Nao e parte do produto, nao e
// distribuido. Existe para que a metrica de paridade seja o comportamento
// real do Obsidian, e nao a nossa interpretacao da documentacao dele.
export default class ParityDumper extends Plugin {
  async onload() {
    this.addCommand({
      id: "dump-metadata-cache",
      name: "Dump metadata cache to JSON",
      callback: async () => {
        const out: Record<string, unknown> = {};

        for (const file of this.app.vault.getMarkdownFiles()) {
          const cache = this.app.metadataCache.getFileCache(file);
          if (!cache) continue;

          out[file.path] = {
            headings: (cache.headings ?? []).map((h) => ({
              level: h.level,
              heading: h.heading,
            })),
            tags: (cache.tags ?? []).map((t) => t.tag.replace(/^#/, "")),
            frontmatterTags: cache.frontmatter?.tags ?? null,
            aliases: cache.frontmatter?.aliases ?? null,
            blocks: Object.keys(cache.blocks ?? {}),
            links: (cache.links ?? []).map((l) => ({
              link: l.link,
              displayText: l.displayText,
              resolved: this.resolve(l.link, file),
            })),
            embeds: (cache.embeds ?? []).map((e) => ({
              link: e.link,
              resolved: this.resolve(e.link, file),
            })),
          };
        }

        await this.app.vault.adapter.write(
          "metadata.json",
          JSON.stringify(out, null, 2),
        );
        new Notice("metadata.json gravado na raiz do cofre");
      },
    });
  }

  private resolve(link: string, from: TFile): string | null {
    const target = this.app.metadataCache.getFirstLinkpathDest(
      link.split("#")[0],
      from.path,
    );
    return target ? target.path : null;
  }
}
```

`tools/parity-dumper/README.md` documenta o procedimento: copiar a pasta para `<cofre-de-teste>/.obsidian/plugins/parity-dumper/`, compilar com `esbuild`, habilitar o plugin, rodar o comando, mover `metadata.json` para `testdata/parity/`. É manual e infrequente por design — o plugin roda uma vez, não a cada build.

- [ ] **Step 2: Montar o corpus e gerar a referência**

`testdata/parity/vault/` recebe 500 notas contendo todos os casos de borda de `testdata/parser/`, mais casos que só existem em escala: colisões de nome entre pastas, aliases colidentes, âncoras para headings renomeados, embeds de anexos, links para notas inexistentes.

Gere a maior parte com `scripts/gen_vault.ps1` de forma determinística e escreva à mão as dezenas de casos de borda. Abra o cofre no Obsidian, espere a indexação terminar, rode o comando do plugin, mova o `metadata.json` para `testdata/parity/`.

- [ ] **Step 3: Escrever o teste de paridade**

```go
func TestParityWithObsidian(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "parity", "vault")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Skip("corpus de paridade ausente; ver tools/parity-dumper/README.md")
	}

	ref := loadReference(t, filepath.Join("..", "..", "testdata", "parity", "metadata.json"))

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// A comparacao e assimetrica de proposito: nossa saida precisa CONTER
	// tudo o que o Obsidian encontrou. O inverso nao e exigido, porque
	// ancoras quebradas e formas de resolucao sao informacao que ele nao expoe.
	// Divergencia para menos e falha; para mais, e o produto.
	for path, want := range ref {
		note, ok := idx.Get(vault.CanonicalPath(path))
		if !ok {
			t.Errorf("%s: ausente do nosso indice", path)
			continue
		}
		assertHeadingsContain(t, path, note.Headings, want.Headings)
		assertTagsContain(t, path, note.Tags, want.Tags)
		assertBlocksContain(t, path, note.Blocks, want.Blocks)
		assertLinksMatch(t, path, note.Links, want.Links, want.Embeds)
	}
}
```

Cada `assert*` reporta a divergência com caminho, o que o Obsidian viu e o que nós vimos. Uma mensagem de falha que diz apenas "divergência em N.md" custa mais tempo do que economiza.

- [ ] **Step 4: Rodar e fechar as divergências**

Run: `go test -race ./internal/index/ -run TestParity -v`

Esperado na primeira execução: divergências. Trate cada uma como bug do parser ou da resolução, **não** ajuste o teste. Casos que o Obsidian trata de forma que consideramos errada devem ser documentados em `tools/parity-dumper/README.md` com o motivo, e só então excluídos.

Critério de saída: divergência zero.

**Casos de frontmatter levantados na revisão da Task 12, a confirmar contra o Obsidian antes de decidir:**

| Entrada | Nosso comportamento hoje | Pergunta |
|---|---|---|
| Delimitador de fechamento com espaço final (`--- `) | Não reconhece frontmatter; o arquivo inteiro vira corpo, em silêncio | O Obsidian tolera o espaço? Se sim, `TrimRight(line, " 	
")` nos dois delimitadores |
| Chave duplicada (`tags:` duas vezes) | `yaml.v3` rejeita o mapa inteiro; `FrontmatterErr` preenchido, tags/aliases/title somem | O Obsidian aceita e usa o último valor. Se confirmado, decodificar com tolerância a duplicata |

As duas produzem a mesma classe de falha que o BOM: uma nota que o Obsidian indexa normalmente perde metadados aqui, sem erro visível ao usuário. Nenhuma deve ser "corrigida" por palpite — a referência de paridade é que decide.

- [ ] **Step 5: Commit**

```bash
git add tools/parity-dumper testdata/parity internal/index
git commit -m "test(index): parity against real Obsidian metadata cache"
```

---

