export const meta = {
  name: 'm2-task-loop',
  description: 'Executa uma tarefa do plano gobsidian: implementa, revisa por tres lentes independentes, corrige Critical/Important, re-revisa.',
  whenToUse:
    'Quando for executar UMA tarefa numerada de docs/superpowers/plans/2026-07-25-gobsidian-v01.md ' +
    '(hoje, as Tasks 33-42 do M2.1). Passe o numero em args, por exemplo args: 35. ' +
    'As tarefas sao estritamente sequenciais — rode uma, feche no ledger, comece a proxima. ' +
    'Nao use para as tarefas que a revisao marcou como do modelo principal (33, 34, 36, 37, 42) ' +
    'sem passar model explicitamente.',
  phases: [
    { title: 'Implementar', detail: 'um agente transcreve o brief e commita' },
    { title: 'Revisar', detail: 'tres lentes independentes: contrato, cobertura, regras do projeto' },
    { title: 'Corrigir', detail: 'so Critical e Important' },
    { title: 'Re-revisar', detail: 'confere as correcoes, nao o resto' },
  ],
}

// ---------------------------------------------------------------------------
// Entrada
// ---------------------------------------------------------------------------

const task = typeof args === 'object' && args !== null ? args.task : args
if (!task) {
  throw new Error('Passe o numero da tarefa em args. Ex.: args: 35, ou args: {task: 35, model: "sonnet"}')
}
const model = typeof args === 'object' && args !== null ? args.model : undefined

const PLAN = 'docs/superpowers/plans/2026-07-25-gobsidian-v01.md'
const BRIEF = `.superpowers/sdd/2026-07-25-gobsidian-v01/task-${task}-brief.md`
const REPORT = `.superpowers/sdd/task-${task}-report.md`

// As regras que valem para todo agente deste workflow. Ficam aqui e nao no
// brief porque o brief e o contrato da tarefa; isto e o contrato do ambiente.
const AMBIENTE = `
Projeto: gobsidian, servidor MCP em Go, Windows. Leia CLAUDE.md antes de agir.

NUNCA rode git checkout, git restore, git stash, git clean nem git reset.
Ha trabalho nao commitado neste repositorio e um subagente ja destruiu trabalho
exatamente assim. Para desfazer o que voce escreveu, edite de volta ou apague o
arquivo especifico que voce criou.

NUNCA rode go mod tidy. Varias dependencias estao fixadas sem importador ainda.

O arquivo .superpowers/sdd/2026-07-25-gobsidian-v01/task-${task}-base.txt esta
sujo de proposito: o primeiro commit da tarefa o recolhe. Nao o commite sozinho.

Verde obrigatorio antes de qualquer commit:
  go test -race ./... ; go vet ./... ; gofmt -l . ;
  GOOS=linux go vet ./... ; GOOS=darwin go vet ./...
`.trim()

const ACHADO_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['findings', 'verdict'],
  properties: {
    verdict: {
      type: 'string',
      enum: ['APPROVED', 'CHANGES_REQUESTED'],
      description: 'CHANGES_REQUESTED se houver qualquer achado Critical ou Important.',
    },
    findings: {
      type: 'array',
      maxItems: 30,
      items: {
        type: 'object',
        additionalProperties: false,
        required: ['severity', 'file', 'line', 'problem', 'evidence', 'fix'],
        properties: {
          severity: { type: 'string', enum: ['Critical', 'Important', 'Minor'] },
          file: { type: 'string' },
          line: { type: 'integer' },
          problem: { type: 'string', description: 'Uma frase. O defeito, nao a impressao.' },
          evidence: {
            type: 'string',
            description:
              'Comando rodado e saida real, ou o cenario concreto de falha: entrada -> resultado errado. ' +
              '"Parece fragil" nao e evidencia.',
          },
          fix: { type: 'string', description: 'A correcao concreta.' },
        },
      },
    },
  },
}

// ---------------------------------------------------------------------------
// 1. Implementar
// ---------------------------------------------------------------------------

phase('Implementar')
log(`Task ${task}: implementando a partir de ${BRIEF}`)

await agent(
  `
${AMBIENTE}

Leia ${BRIEF} — ele e autocontido e carrega tudo: onde a tarefa encaixa, as
decisoes fechadas que a vinculam, a evidencia MEDIDA do defeito que ela corrige,
as armadilhas ja pagas que se aplicam, as verificacoes alem dos passos, as regras
de execucao e o contrato de relatorio. Nao peca contexto adicional; ele nao existe
em lugar nenhum alem do brief e do CLAUDE.md.

O plano e a fonte. Transcreva o desenho da secao; nao improvise uma variante. Se
o codigo do plano nao compilar, corrija o erro mecanico e diga exatamente o que
mudou. Se um teste falhar por motivo que a secao nao explica, PARE e reporte —
nao ajuste a expectativa para o codigo passar.

Toda prova de mutacao que o brief pedir usa a ferramenta, nao a mao:

  pwsh -File scripts/mutate.ps1 -Path <arquivo> -Anchor '<texto exato>' \
       -Replacement '<texto>' -Test <NomeDoTeste> -Package ./internal/x/

Codigo de saida INVERTIDO de proposito: 0 = o teste reprovou sob mutacao (a regra
esta verificada, e o que voce quer); 1 = o teste passou (a regra esta escrita, nao
verificada — escreva o teste que falta); 2 = inconclusivo (ancora ambigua, ou a
mutacao quebrou o build). Cole a saida real dele no relatorio.

Grave o relatorio em ${REPORT}, na forma que o brief exige. NAO escreva prova de
mutacao no condicional ("se removermos X, o teste falharia") — isso nao e prova, e
uma delas ja estava factualmente errada neste projeto. Passado, com saida colada.

Nao escreva numero que voce nao mediu. Onde nao mediu, escreva "nao medido".
Se alguma parte nao deu para fazer, entregue o resto inteiro e diga o que ficou de
fora e por que. BLOCKED com motivo e melhor que entrega que parece completa.

Commit em Conventional Commits, em ingles, com a mensagem que o brief indica.
`.trim(),
  { label: `impl:task-${task}`, phase: 'Implementar', model },
)

// ---------------------------------------------------------------------------
// 2. Revisar — tres lentes, deliberadamente diferentes
// ---------------------------------------------------------------------------
// Redundancia acha o mesmo defeito tres vezes. Lentes distintas acham defeitos
// que a outra nao ve: a revisao do M2 aprovou codigo cujo teste nao conseguia
// reprovar, porque leu o teste em vez de mutar a regra.

const LENTES = [
  {
    key: 'contrato',
    prompt: `
Lente CONTRATO. Compare a entrega com ${BRIEF}, item a item.

Para CADA verificacao que o brief lista em "Verificacoes alem dos passos": ela foi
feita, e o relatorio traz o resultado REAL, ou traz uma afirmacao? Marque como
achado toda verificacao respondida com argumento em vez de medicao.

Para CADA item que o brief manda entregar (arquivos, testes nomeados, docs):
existe no disco? Confira, nao acredite no relatorio.

Escopo encolhido em silencio e achado Important, mesmo que o resto esteja perfeito.
Um schema ou contrato que declara algo que o codigo nao implementa e Critical:
quem le o contrato decide com base nele.
`,
  },
  {
    key: 'cobertura',
    prompt: `
Lente COBERTURA. A pergunta nao e "os testes passam", e "algum teste consegue
reprovar".

Rode primeiro:
  pwsh -File scripts/audit_reports.ps1 ${task}

Depois, para CADA regra que a tarefa reivindica cobrir, rode a mutacao de verdade:
  pwsh -File scripts/mutate.ps1 -Path <arquivo> -Anchor '<texto>' \
       -Replacement '<texto>' -Test <Teste> -Package ./internal/x/

Saida 1 significa que a regra esta escrita e nao verificada: e achado Important,
e a evidencia e a saida do script. Saida 2 significa que voce mutou errado — ajuste
a mutacao, nao registre achado.

Procure especificamente:
- Teste de mecanismo de recuperacao que deixa o caminho normal ligado. Ele mede o
  caminho normal. Foi assim que um requisito P0 ficou com cobertura zero aqui.
- Fixture inerte: a fixture consegue distinguir os dois estados, ou o filtro a
  descartaria de qualquer jeito?
- Guarda que checa existencia quando deveria checar conteudo.
- Valor zero que significa duas coisas: "ausente" e "medido como zero".
- Campo de API com valor fixo, que nao pode estar errado porque nao pode variar.
`,
  },
  {
    key: 'regras',
    prompt: `
Lente REGRAS DO PROJETO. Leia CLAUDE.md e confira o diff contra ele.

- ctx onde ha espera real, e RESPEITADO no corpo. context.Background() dentro de
  internal/ e Critical. ctx que nenhum corpo verifica deve chamar-se _.
- stdout pertence ao JSON-RPC. Todo log em stderr via log/slog. fmt.Println
  alcancavel de serve corrompe a sessao.
- Nenhum tipo do SDK MCP fora de internal/mcpsrv. Nenhum pacote sob internal/ ou
  cmd/ importando net ou net/*.
- Direcao das camadas: internal/service e o unico que enxerga todos os subsistemas.
  Pacote abaixo dele importando service e inversao — ja aconteceu em counters.go.
- Anexo indexado por nome, nunca lido. Arquivo somente-nuvem nunca aberto. Quem
  roda ANTES do guarda precisa do mesmo guarda.
- Chave derivada calculada em dois lugares diverge. Passa por uma funcao so.
- Codigo de plataforma atras de build tag, em arquivo separado. Nada de
  if runtime.GOOS == em logica compartilhada.
- Saida de console em ASCII puro: [OK] [*] [!] [i] [...]
- Sem helpers.go, utils.go, common.go.
- Erro engolido: rode golangci-lint run ./... e confira golangci-lint version
  (exige v2.12.2; v1.x nem carrega o config).
- Comentario de deliberacao commitado. Procure por: "Wait," "For the sake of"
  "we can let it be" "Actually" "TODO". Comentario explica por que o codigo e
  assim; raciocinio sobre o que fazer nao e comentario.
`,
  },
]

const revisoes = await parallel(
  LENTES.map((lente) => () =>
    agent(
      `${AMBIENTE}\n\nVoce esta REVISANDO a Task ${task}, ja implementada e commitada.\n` +
        `Plano: ${PLAN}. Brief: ${BRIEF}. Relatorio: ${REPORT}.\n` +
        `Veja o diff com: git diff $(cat .superpowers/sdd/2026-07-25-gobsidian-v01/task-${task}-base.txt)..HEAD\n\n` +
        lente.prompt.trim() +
        `\n\nNAO altere codigo. Reporte.\n` +
        `Toda evidencia e comando rodado com saida real, ou cenario concreto entrada -> resultado errado.\n` +
        `"Parece fragil" e "poderia ser melhor" nao sao achados. Se nao achou nada, devolva lista vazia —\n` +
        `revisao que inventa achado para parecer util custa mais que revisao silenciosa.`,
      { label: `review:${lente.key}`, phase: 'Revisar', schema: ACHADO_SCHEMA },
    ),
  ),
)

const achados = revisoes
  .filter(Boolean)
  .flatMap((r) => r.findings || [])

const bloqueantes = achados.filter(
  (f) => f.severity === 'Critical' || f.severity === 'Important',
)

log(
  `Task ${task}: ${achados.length} achado(s), ${bloqueantes.length} bloqueante(s) ` +
    `(${achados.filter((f) => f.severity === 'Minor').length} Minor diferidos).`,
)

if (bloqueantes.length === 0) {
  log(`Task ${task}: aprovada pelas tres lentes. Registre no ledger antes de dizer que acabou.`)
  return {
    task,
    verdict: 'APPROVED',
    findings: achados,
    proximoPasso:
      `pwsh -File scripts/sdd.ps1 review ${task} e registrar no ledger ` +
      `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`,
  }
}

// ---------------------------------------------------------------------------
// 3. Corrigir — so Critical e Important
// ---------------------------------------------------------------------------
// Minor vai para a lista de diferidos do ledger. Corrigir Minor no meio de um
// fix pass mistura o diff e a re-revisao deixa de conseguir olhar so as
// correcoes.

phase('Corrigir')

const listaFix = bloqueantes
  .map(
    (f, i) =>
      `${i + 1}. [${f.severity}] ${f.file}:${f.line}\n` +
      `   Problema: ${f.problem}\n` +
      `   Evidencia: ${f.evidence}\n` +
      `   Correcao proposta: ${f.fix}`,
  )
  .join('\n\n')

await agent(
  `
${AMBIENTE}

Fix pass da Task ${task}. Corrija EXATAMENTE os achados abaixo, nada alem deles.

${listaFix}

Antes de corrigir cada um: confirme que o achado procede. Revisor tambem erra, e
neste projeto ja houve achado que a revisao aceitou e estava errado — inclusive um
em que a revisao concordou com a especificacao, e a especificacao e que estava
errada. Se um achado nao procede, NAO o implemente: diga por que, com a evidencia
que o refuta.

Se a causa do defeito estiver no PLANO e nao na transcricao, corrija o plano em
${PLAN} tambem, e commite essa correcao. O proximo implementador transcreve do
plano, e um plano nao corrigido reinjeta o defeito.

Cada regra que voce corrigir ganha prova de mutacao com scripts/mutate.ps1, com a
saida colada. Atualize ${REPORT} com o que mudou no fix pass.
`.trim(),
  { label: `fix:task-${task}`, phase: 'Corrigir', model },
)

// ---------------------------------------------------------------------------
// 4. Re-revisar — so as correcoes
// ---------------------------------------------------------------------------

phase('Re-revisar')

const rerevisao = await agent(
  `
${AMBIENTE}

Re-revisao da Task ${task}. Olhe SO se os achados abaixo foram fechados, e se o
fix pass introduziu regressao. Nao re-revise o resto da tarefa.

${listaFix}

Para cada achado: fechado, nao fechado, ou recusado com justificativa? Achado
recusado com evidencia que o refuta e resultado valido — registre como tal, nao
como pendencia.

Rode e cole a saida:
  pwsh -File scripts/audit_reports.ps1 ${task}
  pwsh -File scripts/verify.ps1

Para cada regra que o fix pass alega ter coberto, rode scripts/mutate.ps1 voce
mesmo. Nao aceite a saida colada por quem corrigiu — este projeto ja teve prova de
mutacao escrita no condicional, e uma delas estava factualmente errada.
`.trim(),
  { label: `re-review:task-${task}`, phase: 'Re-revisar', schema: ACHADO_SCHEMA },
)

const restantes = (rerevisao?.findings || []).filter(
  (f) => f.severity === 'Critical' || f.severity === 'Important',
)

return {
  task,
  verdict: restantes.length === 0 ? 'APPROVED' : 'CHANGES_REQUESTED',
  achadosOriginais: achados,
  aposFixPass: rerevisao?.findings || [],
  minorDiferidos: achados.filter((f) => f.severity === 'Minor'),
  proximoPasso:
    restantes.length === 0
      ? `pwsh -File scripts/sdd.ps1 review ${task}, registrar no ledger, e gravar a base da proxima tarefa`
      : `${restantes.length} bloqueante(s) sobrando — rode o workflow de novo ou trate a mao`,
}
