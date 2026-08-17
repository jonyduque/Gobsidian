---
title: Achados em aberto
type: note
status: active
description: Defeitos e oportunidades levantados pela revisão de 2026-08-15, ainda não corrigidos.
source_paths:
- docs/REVISAO-2026-08-15.md
- docs/superpowers/plans/2026-08-16-revisao-fixes.md
- scripts/check_tool_params.ps1
source_commit: b2be4925
tags:
- revisao
- pendencias
language: pt-BR
updated_at: '2026-08-16'
---

# Achados em aberto

Índice do que a revisão de 2026-08-15 encontrou e **ainda não foi corrigido**. O
detalhe de cada item, com arquivo e linha, está em
`docs/REVISAO-2026-08-15.md` — esta página só orienta.

> **Nada aqui foi commitado como correção.** Se você está lendo antes de mexer no
> código, confira se o achado ainda existe: `git log` pode ter passado por cima.

O plano que ataca estes achados é
`docs/superpowers/plans/2026-08-16-revisao-fixes.md` — Tasks 104 a 123, em quatro
fases com brief autocontido, mais cinco fases nomeadas sem brief. Ele fecha oito
decisões (D-R-1 a D-R-8) que **não se re-litigam** dentro das tarefas.

As onze tarefas de tier barato estão prontas para despacho: código de teste
completo, a seção "Ambiente de teste" que diz em que **pacote** cada arquivo
fica, e o prompt literal de delegação.

## Críticos

| # | Achado | Onde |
|---|---|---|
| 1 | **Escrita não passa pelo confinamento de caminho.** `checkWriteAllowed` é uma checagem própria; `CanonicalPath` é construída por conversão. No Windows, `..\..\x.md` escapa da raiz do cofre | `internal/service/write.go:79` |
| 2 | **O laço de boot do índice de busca abre placeholder de nuvem.** Ele não passa por `Inverted.Update`, onde a guarda mora — e o teste que "prova" a regra exercita o caminho que a produção não usa | `cmd/gobsidian/serve.go:323` |
| 3 | **`expected_hash` compara contra o hash do índice, não dos bytes lidos.** Na janela do debounce, uma edição externa passa despercebida e é sobrescrita | `service/write.go:172,278` |
| 4 | **Symlink no cofre lê e escreve fora dele.** Resolvível com `os.Root` — disponível desde Go 1.24 | `vault/walk.go`, `vault/path.go` |

## Importantes, por tema

**Contratos que mentem.** `tag_list.sort` e `tag_list.hierarchical` declarados e
nunca lidos; `max_results` é cadeia de config morta; filtro de data descartado em
<!-- wiki-refs: ignore max_results -- o achado E que este nome de schema nao existe no codigo -->
silêncio quando não parseia; `Truncated` marca verdadeiro quando o recorte mede
exatamente `max_bytes`.

**Escrita.** `note_move` copia e remove em vez de renomear, com o erro do remove
descartado; não há rollback das notas citantes já reescritas; `WriteAtomic` não
faz fsync do diretório e perde o modo do arquivo original.

**Daemon.** `acceptLoop` encerra para sempre num erro transitório de `Accept`;
`ipc.Listen` desvincula o socket sem checar se há daemon vivo; `CloseWrite` é
declarado, verificado no handshake e **nunca chamado**; flags da segunda ponte
viram no-op silencioso.

Correção ao relatório: ele diz que *"o lock não é a causa: o `Listen` é"*, e isso
é exclusão indevida — a janela do lock explica o **lançamento** do segundo daemon
e o unlink explica a **tomada do nome**. As duas entram. Detalhe e evidência
(`internal/daemon/lock.go:65-79`) na Fase 4 do plano.

**Diagnóstico.** `FrontmatterErr` é preenchido pelo parser e **nunca lido** por
ninguém; `index.Build` descarta arquivo ilegível sem log nem contador.

**Deliberação commitada, quatro pontos.** Dois a revisão não viu, porque não
olhou `_test.go` nem comentário de histórico. `internal/service/read.go:244`:
`// better condition`. `internal/index/note.go:142`: andaime de mutação rodando
em produção, uma vez por nota indexada. `internal/service/read_test.go:30`: duas
perguntas em inglês dentro de `newTestService`, o helper que todo o lote usa.
`internal/service/graph.go:401`: comentário de histórico que o `git log` já sabe.
Todos na Task 123.

**Gates que passam verdes por cima do defeito.** `scripts/check_tool_params.ps1`
existe para pegar exatamente os "contratos que mentem" acima e, em `b2be492`,
saía **`0`** com os campos mortos: casava nome de campo no pacote inteiro, não
por struct, então `tag_list.sort` passava porque outra tool tem um `Sort` que é
lido. E `check_doc_refs`, `check_readme_anchors` e `check_tool_params` **não
estão no CI** — só existem dentro do `verify.ps1`, que nenhum job invoca.

Precisão que a revisão não fez: são **dois** campos de schema mortos, não três.
`max_results` também é cadeia morta, mas é flag de CLI e nunca esteve numa
<!-- wiki-refs: ignore max_results -- nome de flag de CLI, nunca foi campo de schema; a frase existe para dizer isso -->
struct `*Input` — está fora do alcance deste instrumento por construção, e é a
Task 120 que responde por ele.

> **Estado em 2026-08-17, ainda não commitado.** A Task 104 está **feita**, na
> segunda tentativa. O script passou a decidir o nível 1 só por `$pVar.$Campo`
> com `-cmatch`, sem disjunto de nome nu, e struct sem handler resolvido virou
> achado (`HANDLER-NAO-RESOLVIDO`) em vez de cair para o pacote inteiro. A
> revisão re-executou quatro mutações próprias, todas com SHA-256 idêntico
> antes e depois: apagar o leitor de `noteListInput.Sort` ou de
> `moveInput.UpdateLinks` leva o gate de 2 para 3; o domínio parando de ler
> `MetadataRequest.Include` acusa `noteMetadataInput.Include`; e o handler de
> `tag_list` recebendo ponteiro acusa os quatro campos dele.
>
> **No HEAD o gate sai `1` com DOIS campos** — `tagListInput.Sort` e
> `tagListInput.Hierarchical` — e assim fica até a Task 120 implementar ou
> remover os dois. O relatório da tarefa mostra três; aqueles três são o estado
> **sob mutação**, não o HEAD.
>
> Pendência registrada: o nível 2 casa `.Campo` em qualquer lugar de
> `internal/`, sem escopo da struct de destino, então não distingue
> `TagRequest.Sort` de `ListRequest.Sort`. O limite está escrito no cabeçalho do
> script; fechá-lo exige seguir o campo até o tipo de destino.

**Ferramenta.** `netcheck` não enxerga métodos de `net.Dialer`; o CI não roda o
<!-- wiki-refs: ignore net.Dialer -- API que o projeto NAO usa; e justamente o buraco descrito -->
cenário `daemon-idle`; `ResolvePath` faz varredura O(N) num mapa onde a chave é
única, e `ErrAmbiguousPath` por esse ramo é inalcançável.

## Da experiência de uso real

Uma sessão real fez o modelo **abandonar o gobsidian e usar outro servidor MCP**
para ler uma nota de 255 KB convertida de livro. Os cinco defeitos que a
causaram estão em
[Incidente de campo de 2026-08-15](incidente-de-campo-2026-08-15.md), com a
transcrição e o mapeamento para as tarefas que os corrigem.

Dois da mesma superfície, fora do incidente:

- **Normalização Unicode ausente nas chaves do índice.** `publishNameLocked`,
  `aliasKey`, `nomeChave` e `ResolvePath` aplicam só caixa, enquanto
  `text.Normalize` e `parser.Slug` aplicam NFC/NFD. `Capítulo` em NFD e em NFC
  são strings diferentes. Aparece em cofre sincronizado com macOS. (Task 114.)
- **A carga preguiçosa bloqueia sem prazo, e `INDEX_BUILDING` é inalcançável.**
  `internal/service/search_lazy.go:33` trava num mutex não cancelável, e o
  carregador inclui `buildInvertedIndex` — a **primeira** `vault_search` paga a
  tokenização inteira do cofre. Contradiz o comentário de
  `cmd/gobsidian/servico.go:105`. (Task 117.)

## Desempenho — hipóteses, não medições

Os itens de desempenho da revisão têm mecanismo identificado e **não foram
medidos**. Os principais: pontuar o BM25 em espaço de IDs densos (elimina
milhares de mapas por consulta e resolve o `avgdl` O(N) de brinde);
`Heading.Slug` é pré-computado, persistido no cache e **nunca lido** — três
lugares recomputam.

## O que foi medido naquele dia

Está em [Medições de 2026-08-15](medicoes-2026-08-15.md), com as condições, as
regras de decisão declaradas antes de medir, e o que continua sem medição. Em
uma linha: o toolchain 1.26 **não** move os benchmarks de forma relevante, e a
rejeição do `GOGC` continua de pé.

## Ver também

- `docs/REVISAO-2026-08-15.md` — o relatório completo, com a ordem de ataque.
- `docs/superpowers/plans/2026-08-16-revisao-fixes.md` — o plano que os ataca,
  com as decisões fechadas, os tiers e o prompt de despacho.
- [Medições de 2026-08-15](medicoes-2026-08-15.md) — a única parte medida da
  revisão.
- [Incidente de campo de 2026-08-15](incidente-de-campo-2026-08-15.md) — a falha
  observada em produção, e o que a causou.
