### Task 26: Fechamento da v0.1

#### Onde isto encaixa

Última tarefa de M1. Fecha a v0.1: a suíte inteira, a medição contra o orçamento de performance, a documentação de operação e o release.

#### O que já está fechado e vincula esta tarefa

- **`scripts/test_orphans.ps1 -Cycles 100` com zero órfãos é critério de bloqueio de release**, não meta de esforço. Se falhar, esse é o resultado mais valioso disponível: diagnostique com `--log-level debug` e reporte qual mecanismo disparou ou deixou de disparar. **Nunca enfraqueça o teste para ele passar.**
- **A verificação de rede do CI inspeciona nossos pacotes**, não o fecho transitivo. `net/http` chega pelo SDK e é esperado.
- **O binário é estático:** `CGO_ENABLED=0`. Faz parte do requisito de instalação trivial.
- **Saída de console em ASCII puro.** Console PowerShell em CP-850 renderiza o resto como lixo.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **`golangci-lint` local verde não significa CI verde.** O `go.mod` declara `go 1.25.0`, e um binário compilado com Go mais antigo recusa o config antes de analisar linha nenhuma. O CI fixa a versão de propósito; confira `golangci-lint version` antes de confiar num zero.
- **Alvo não medido é ficção.** Se um número do orçamento de performance estourar, registre o valor real em `docs/OPERACAO.md` em vez de silenciar. Alvo não atingido e documentado é informação.
- **Documentação que promete o que o código não faz.** Ao atualizar o README e o PRD, descreva o que a v0.1 **não** faz — sem watcher, sem busca, sem escrita, e reindexar exige reiniciar o host.

#### Verificações além dos passos

- Toda a suíte verde nos três alvos, mais `check_net.ps1`, `build.ps1` e os 100 ciclos de órfãos.
- Os números medidos estão em `docs/OPERACAO.md` com data e máquina?
- O README descreve as limitações da v0.1, não só as capacidades?

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o código desta seção; não improvise uma variante. Se ele não compilar, corrija o erro mecânico e **diga exatamente o que mudou**. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar. Teste dobrado para passar é como defeito silencioso chega em produção.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** Há trabalho não commitado de outras frentes neste repositório, e um subagente já destruiu trabalho exatamente assim. Para desfazer o que você escreveu, edite de volta ou apague o arquivo específico que você criou.
- **`go mod tidy` está proibido.** Várias dependências fixadas ainda não têm importador, e o `tidy` as removeria — inclusive o pin obrigatório do SDK de MCP. Se o build reclamar de entrada faltando em `go.sum`, **pare e reporte**; não rode `go get`.
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.** Escrita em modo texto converte o arquivo inteiro para CRLF no Windows e o `gofmt` rejeita. Já custou dois commits neste projeto.
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês. Sem arquivos chamados `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave o relatório completo em `.superpowers/sdd/task-26-report.md`, com: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a tabela de verificações extras acima com o resultado real de cada uma; arquivos alterados; achados da auto-revisão; correções mecânicas que fez no código do plano; e preocupações.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, as respostas diretas pedidas acima, e preocupações. O detalhe mora no arquivo de relatório, não na resposta.

**Files:**
- Modify: `README.md` (seção Estado), `docs/PRD.md` (marcar M0 e M1)
- Create: `.github/workflows/release.yml`
- Create: `docs/OPERACAO.md`

**Interfaces:**
- Consumes: tudo
- Produces: tag `v0.1.0` com binários para Windows, macOS e Linux

- [ ] **Step 1: Rodar a suíte completa**

```powershell
go vet ./...
go test -race ./...
golangci-lint run
.\scripts\check_net.ps1
.\scripts\build.ps1
.\scripts\test_orphans.ps1 -Cycles 100
```

Todos precisam passar. Se algum falhar, corrija antes de seguir — não anote como pendência.

- [ ] **Step 2: Medir contra o orçamento de performance**

Com o cofre real, meça e registre:

```powershell
Measure-Command { .\bin\gobsidian.exe index --vault $VaultPath --stats }
```

Compare com RNF-01 (indexação a frio ≤ 3 s para 5.000 notas) e RNF-07 (RSS ≤ 60 MB). Estes são os dois alvos que a v0.1 já pode verificar; os demais dependem de busca e cache.

Se estourarem, registre o número real em `docs/OPERACAO.md` em vez de silenciar. Um alvo não atingido e documentado é informação; um alvo não medido é ficção.

- [ ] **Step 3: Escrever `docs/OPERACAO.md`**

Conteúdo mínimo: como registrar no Claude Desktop (referenciando `docs/WINDOWS.md` §8), como diagnosticar quando o servidor não aparece, como interpretar cada campo de `vault_stats`, como ler o log de nível debug, e a tabela de medições do Step 2 com data e máquina.

- [ ] **Step 4: Configurar o release**

`.github/workflows/release.yml` dispara em tag `v*`, compila para `windows/amd64`, `darwin/arm64` e `linux/amd64` com `CGO_ENABLED=0` e as `-ldflags` de versão, e anexa os três binários à release do GitHub.

- [ ] **Step 5: Atualizar o estado nos docs**

Em `README.md`, substitua a seção **Estado** pelo estado real: v0.1 publicada, o que ela faz, o que ainda não faz — sem watcher, sem busca, sem escrita — e que reindexar exige reiniciar o host.

Em `docs/PRD.md` §9, marque M0 e M1 como concluídos com a data.

- [ ] **Step 6: Commit e tag**

```bash
git add README.md docs .github
git commit -m "docs: v0.1 release notes and operations guide"
git tag -a v0.1.0 -m "v0.1.0: lifecycle, parser, index, read tools"
git push origin main --tags
```

**Fim da v0.1.** O produto substitui a parte de leitura do fluxo de trabalho. O que vem depois é construído sobre uma fundação sob uso real, com os defeitos que só o uso real revela já visíveis.

---

## M2 a M6 — escopo em altitude de tarefa

Cada marco recebe seu próprio plano detalhado quando chegar a vez, escrito contra o código que existir então. Planejá-los agora em nível de passo produziria detalhe que envelhece antes de ser usado.

### M2 — Watcher (meia semana)

| Tarefa | Entregável | Critério |
|---|---|---|
| W1 | `watcher/filter.go` — relevância de evento | Eventos fora de `.md`/anexos e em diretórios excluídos descartados |
| W2 | `watcher/debounce.go` — janela e coalescência | N eventos no mesmo caminho em 250 ms produzem um reparse |
| W3 | Verificação de mudança real | mtime e tamanho iguais aos indexados descartam sem parsear |
| W4 | `watcher/overflow.go` — reconciliação | `ErrEventOverflow` dispara varredura completa; nenhuma divergência sobrevive |
| W5 | Correlação de rename por `xxhash` | Remoção e criação na mesma janela com mesmo hash são reportadas como rename, **não** reescritas |
| W6 | Contadores em `vault_stats` | Recebidos, coalescidos, processados, overflows |

Risco principal: rajadas do OneDrive. É o risco mais provável do PRD §8 e o mais chato de diagnosticar — por isso W6 não é opcional e não fica para depois.

### M3 — Busca (1 semana)

| Tarefa | Entregável | Critério |
|---|---|---|
| S1 | `search/analyzer.go` | Indexação dupla: forma crua e reduzida; teste com corpus jurídico real |
| S2 | `search/inverted.go` | Dicionário de termos e posting lists |
| S3 | `search/bm25.go` | k1=1.2, b=0.75; pesos título 3x, headings 2x, corpo 1x |
| S4 | `search/snippet.go` | Trecho com destaque, `snippet_chars` respeitado |
| S5 | Tool `vault_search` | Filtros combináveis; query vazia redireciona para o caminho de metadados |
| S6 | `index/cache.go` e `search/persist.go` | Cabeçalho com `format_version`, `parser_version`, `analyzer_version`; RNF-02 medido |

Q3 do PRD §11 fecha aqui: medir o custo real de reconstruir a busca a partir do índice de metadados já carregado decide se o índice invertido também vai para o cache.

### M4 — Escrita (2 semanas)

| Tarefa | Entregável | Critério |
|---|---|---|
| E1 | `writer/lock.go` | Mutex por caminho; teste com escritas concorrentes na mesma nota |
| E2 | `writer/atomic.go` | Temporário no mesmo diretório, `Sync`, rename com retry; teste de crash injetado, 1.000 iterações |
| E3 | `writer/diff.go` | Myers sobre linhas, ~150 linhas, sem dependência |
| E4 | `writer/section.go` | `note_append` e `note_patch` por heading; EOL e BOM preservados |
| E5 | `writer/block.go` | `replace_block` por `^id` |
| E6 | Tools `note_create`, `note_append`, `note_patch` | `dry_run` e `expected_hash` em todas |
| E7 | `--read-only` verificado | Tools de escrita ausentes de `ListTools`, não apenas rejeitadas |

RNF-11 é o critério de bloqueio deste marco: zero notas corrompidas em 1.000 iterações de crash injetado. É a única parte do produto que pode destruir dados, e vem depois de tudo por isso.

### M5 — Refatoração do cofre (1 semana)

| Tarefa | Entregável | Critério |
|---|---|---|
| R1 | `writer/linkrewrite.go` | Alias, âncora e forma original preservados; `Link.Raw` é o insumo |
| R2 | Tool `note_move` | Falha parcial reporta com precisão o que foi aplicado |
| R3 | Tool `note_delete` | `to_trash` padrão; relatório prévio de links que quebrarão |
| R4 | Âncoras quebradas em `vault_stats` | Já implementado em M1; exposto aqui no relatório de impacto |

### M6 — Endurecimento (1 semana) — v1.0

| Tarefa | Entregável | Critério |
|---|---|---|
| H1 | `bench.yml` no CI | RNF-01 a RNF-08 medidos; regressão acima de 20% falha o build |
| H2 | `gen_vault.ps1` determinístico | Cofre sintético de 5.000 notas reproduzível |
| H3 | Subcomandos `index`, `search`, `inspect` | RF-52 |
| H4 | `go vet -vettool` com `netcheck` no CI | Segunda parte da garantia de RNF-30 |
| H5 | Release v1.0.0 | Binários para os três sistemas |

---

## Auto-revisão do plano

**Cobertura da especificação.** Requisitos do PRD mapeados a tarefas: RF-01, RF-02 e RF-08 → Task 19; RF-03 a RF-05 → M2; RF-06 → M3/S6; RF-07 → M2 (o parsing de `.gitignore` entra com o watcher, que é quem precisa dele); RF-10 a RF-17 → Tasks 12–18; RF-18 → Task 17; RF-19 (callouts, P2) → pós-1.0, não planejado; RF-20 a RF-25 → M3; RF-26 (P2) → Q1 em aberto; RF-30 a RF-38 → M4; RF-35 e RF-36 → M5; RF-40 a RF-44 → Tasks 3–6 e 11; RF-50 e RF-51 → Tasks 9 e 10; RF-52 → M6/H3; RF-53 → Global Constraints e Task 9; RF-54 (P2) → fora da v1 por D7; RF-55 → Tasks 9 e M4/E7; RF-60 a RF-63 → Tasks 8, 19 e 20. RNF-01 e RNF-07 → Task 26/Step 2; RNF-02 a RNF-06, RNF-08, RNF-09 → M6/H1; RNF-10 → Task 11; RNF-11 → M4/E2; RNF-12 → Tasks 19 e 21; RNF-13 → Task 9; RNF-20 a RNF-23 → Tasks 7, 8 e 10; RNF-24 → Task 9; RNF-30 → Task 1 e M6/H4; RNF-31 e RNF-32 → Task 7; RNF-33 → Tasks 9 e M4/E7.

**Lacuna conhecida.** RNF-32 (links simbólicos que apontem para fora do cofre não são seguidos) tem teste apenas indireto na Task 7, porque criar symlink no Windows exige privilégio elevado e o teste falharia em máquina de desenvolvimento comum. `vault.Walk` usa `filepath.WalkDir`, que não segue symlinks por padrão — a propriedade vale por construção. Adicione um teste explícito em M6, marcado com `t.Skip` quando a criação de symlink falhar por permissão.

**Consistência de tipos.** `vault.CanonicalPath` é o tipo de caminho em toda a cadeia — `vault`, `index`, `service` — e nunca vira `string` solto. `parser.Link` é embutido em `index.ResolvedLink`, e não duplicado. `Note.Hash` é `uint64` no índice e string hexadecimal no contrato MCP; a conversão acontece em `service/read.go` e em nenhum outro lugar. `service.Index` é declarado como interface na Task 9 e satisfeito por `*index.Index` na Task 23, o que é o que permite testar o serviço sem construir um índice completo.

**Sem marcadores pendentes.** Nenhum passo contém "TBD", "implementar depois" ou "similar à tarefa N". As Tasks 15 a 17, 20 a 24 descrevem regras em prosa numerada em vez de código completo por escolha: são extensões cujo formato já foi estabelecido integralmente na Task 14, e as regras são o que o implementador precisa saber que os testes não dizem sozinhos. Todo teste está escrito por inteiro.

---

## Referências

| Documento | Papel |
|---|---|
| `docs/PRD.md` | Requisitos, prioridades, decisões fechadas (D1–D13), questões em aberto |
| `docs/ARCHITECTURE.md` | Camadas, fluxos, modelo de dados, decisões arquiteturais (AD-01–AD-09) |
| `docs/ESTRUTURA.md` | Árvore de diretórios, convenções de código, build |
| `docs/TOOLS.md` | Contrato de cada tool: schemas, retornos, códigos de erro |
| `docs/WINDOWS.md` | OneDrive, MAX_PATH, casing, fsnotify, registro no Claude Desktop |
