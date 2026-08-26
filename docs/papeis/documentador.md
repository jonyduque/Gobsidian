# Papel: documentador

Você vai escrever ou atualizar documentação.

**A premissa:** neste projeto a documentação é contrato, não enfeite. O modelo
do outro lado lê o schema e o `TOOLS.md` para decidir o que pedir. Documento que
promete o que o código não entrega custa mais que documento ausente, porque
ninguém desconfia dele.

---

## A regra de ouro

**Duas cópias do mesmo fato divergem, e a menos consultada é a que fica errada.**
É a lição de `byAlias` aplicada a texto, e ela já se realizou aqui: a redação do
RNF-30 foi atualizada no `PRD.md`, no `ARCHITECTURE.md` e no `OPERACAO.md` em
2026-08-05, e ficou velha no `CLAUDE.md` até 2026-08-26 — tempo suficiente para
uma auditoria acusar violação falsa.

Consequência prática: **cite, não recopie**. Se você está prestes a resumir um
documento normativo em outro, ponha um link.

---

## Hierarquia

| Camada | Arquivos | Papel |
|---|---|---|
| **Normativa** | `PRD.md`, `ARCHITECTURE.md`, `TOOLS.md`, `ESTRUTURA.md`, `WINDOWS.md`, `OPERACAO.md` | A especificação. Onde divergir de qualquer outra coisa, **esta vence**. |
| **Operacional** | `CLAUDE.md`, `AGENTS.md`, `docs/papeis/*` | Como trabalhar aqui. Subconjunto executável. |
| **Histórica** | `ARMADILHAS.md`, `ESTADO.md` | O porquê. Defeitos pagos e medições. |
| **Derivada** | `docs/wiki/**` | Explica **o código**. Cita a normativa; não a recopia. |

---

## Não escreva número que você não mediu

`OPERACAO.md` chegou a trazer uma tabela chamada "Resultado da Medição v0.1" com
*"Concluído abaixo do alvo (ex: 408ms em teste local)"* e *"Tende a ficar ~30-45
MB"*. O primeiro é exemplo, o segundo é expectativa; nenhum é medição.

- Alvo não atingido **e registrado** é informação.
- Alvo não medido **apresentado como resultado** é ficção com aparência de
  tabela.
- Se não mediu, escreva **"não medido"**.

Palavras que denunciam hedge apresentado como resultado: *tende a*,
*aproximadamente*, *e.g.*, *ex:*, *deveria*. `scripts/audit_reports.ps1` procura
por elas.

**Não afirme estado que você não verificou.** O README declarou "v0.1 publicada"
sem tag, sem release e sem gate.

---

## Os gates de documentação

Os três rodam dentro de `verify.ps1` e no CI. **Entraram na bateria em
2026-08-11 porque até então não rodavam em lugar nenhum** — três seções do README
ficaram sem link por um marco inteiro por causa disso.

```bash
pwsh -File scripts/check_doc_refs.ps1        # token entre crases que parece codigo e nao existe
pwsh -File scripts/check_readme_anchors.ps1  # toda ancora resolve; toda H2 e alcancavel
pwsh -File scripts/check_tool_params.ps1     # parametro documentado existe no schema
```

**`check_doc_refs` dispensa por linha, não por lista global.** A diretiva é:

```markdown
<!-- check-doc-refs: ignore <tokens> -- <motivo> -->
```

e o **motivo é obrigatório**: sem ele vira `DISPENSA-INVALIDA` e o token
continua acusado. Uma lista global no topo do script dispensaria `helpers.go` em
*todo* documento, inclusive num que passasse a afirmar, errado, que o arquivo
existe. A dispensa mora colada à afirmação que a justifica, e as usadas saem
impressas a cada rodada — lista de exceção que ninguém vê deixa de ser revisada.

---

## Encoding

Os documentos são em português, e **ferramenta que reescreve `.md` pode gravar em
cp1252**. Depois de qualquer reescrita:

```bash
python -c "open('ARQUIVO.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
```

Script Python que edita `.md` versionado precisa de `newline=""` na leitura **e**
na escrita, e `assert` do texto-âncora antes de substituir — `str.replace` que
não casa não falha, segue em silêncio. Detalhe em
[`../ARMADILHAS.md`](../ARMADILHAS.md).

---

## O wiki do codebase (`docs/wiki`)

Camada derivada que explica **o código**. Entrada em `docs/wiki/Home.md`.

Quatro regras:

- **Toda página declara `source_paths`.** É o que liga arquivo alterado a página
  a reescrever. Sem isso o wiki vira prosa solta e o drift deixa de ser
  detectável.
- **Não preencha `source_commit` à mão** — o `post-ingest` carimba o que estiver
  vazio, e nunca sobrescreve o que já tem valor.
- **Não recopie `docs/`.** Página que só resumiria um documento normativo não
  deve existir; ponha um link no `Home.md`.
- **Símbolo citado entre crases tem de existir no código.**

```powershell
# o que mudou desde o ultimo ingest, e que paginas declaram esses arquivos
python <skill>/scripts/wiki_scan.py changes . --wiki docs/wiki

# criar pagina nova (nunca cole frontmatter a mao)
python <skill>/scripts/wiki_new.py . --type feature --title "X" --source internal/x/y.go --apply

# fechar o ciclo: carimbo -> indice -> cobertura -> doctor -> checkpoint
<skill>/scripts/wiki-pipeline.ps1 post-ingest -Repo . -AdvanceCheckpoint

# saude, incluindo referencias a codigo que nao resolvem
<skill>/scripts/wiki-pipeline.ps1 lint -Repo .
```

`<skill>` é `~/.claude/skills/codebase-wiki`, e ela carrega sozinha nos gatilhos
("atualizar o wiki", "documentar o codebase").

O wiki **não** entra no `verify.ps1` nem no CI: ele documenta, não gateia. Quem
o mantém alinhado é o `post-ingest` rodado junto da mudança que o tornou
desatualizado.

Ao responder algo sobre o projeto a partir do wiki, **cite a página e o caminho
de código** (`internal/service/write.go:79`), e verifique no código quando a
página estiver `status: stale`. Página defasada é ponto de partida, não resposta.

---

## O hook de commit

`.claude/settings.json` instala um hook `PreToolUse` filtrado para
`git commit*`, que roda `scripts/pre_commit_docs.ps1`. Ele inspeciona o que está
em stage e **pergunta** (não bloqueia) quando há `.go` de produção sem
documentação nem ledger junto.

A decisão é `ask` de propósito: gate que reprova sozinho e sem recurso ensina a
contornar o gate. Quem decide é a pessoa, com a lista do que provavelmente
ficou para trás na tela.
