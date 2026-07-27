# gobsidian

MCP server em Go. Expõe cofre local Obsidian a hosts MCP, roda como subprocesso sobre stdio.

Spec vive em `docs/`: `PRD.md` (requisitos + decisões fechadas D1–D13), `ARCHITECTURE.md` (camadas + AD-01–AD-09), `ESTRUTURA.md` (árvore + convenções), `TOOLS.md` (contrato de cada tool), `WINDOWS.md` (OneDrive, MAX_PATH, casing). Plano de implementação em `docs/superpowers/plans/2026-07-25-gobsidian-v01.md` — fonte de onde código é transcrito. Plano e código não podem divergir.

**Docs em português. Ferramenta que reescreve `.md` pode gravar em cp1252.** `caveman-compress` fez isso com este arquivo: prosa saiu cp1252, bloco de código saiu UTF-8 — arquivo misto, `Expõe` virou `Exp\xf5e`. Nada se perde (cp1252 é reversível, ao contrário de `U+FFFD`), mas transcodificar o arquivo inteiro de uma vez duplica os acentos das regiões que já eram UTF-8.

Depois de qualquer ferramenta que reescreva um `.md`, confira:

```bash
python -c "open('CLAUDE.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
```

Reparo de arquivo misto: decodificar byte a byte, tentando sequência UTF-8 válida primeiro e caindo pra cp1252 onde falhar. Conferir palavras-sonda acentuadas depois, inclusive uma dentro de código inline.

## Comandos

```bash
go test -race ./...          # a versão sem -race não conta
go vet ./...
gofmt -l .
GOOS=linux go vet ./...      # e darwin: há arquivos com build tag
pwsh -File scripts/check_net.ps1
pwsh -File scripts/build.ps1
pwsh -File scripts/test_orphans.ps1 -Cycles 100
golangci-lint run ./...          # exige v2.12.2; v1.x nem carrega o config
```

**`golangci-lint` local verde não significa CI verde.** O `go.mod` declara `go 1.25.0`, e um binário compilado com Go mais antigo recusa o config antes de analisar linha nenhuma: `can't load config: the Go language version (go1.24) used to build golangci-lint is lower than the targeted Go version (1.25.0)`. O CI fixa `v2.12.2` de propósito — com a versão flutuando, os dois lados resolvem binários diferentes e a checagem local para de dizer qualquer coisa sobre o pipeline. Confira `golangci-lint version` antes de confiar num zero.

## Dependências

**Nunca rode `go mod tidy`.** Várias deps fixadas mas sem importador ainda — `goldmark`, `yaml.v3`, `x/text` só ganham um em M1 e M3. `tidy` removeria elas, junto com pin do SDK MCP, que é decisão fechada (PRD D6).

Quando pacote importa dep pela primeira vez, `go.sum` pode faltar entradas transitivas. Comando certo: `go get <caminho-do-pacote>@<versão-fixada>` — caminho do **pacote**, não do módulo. `go get <módulo>@<versão>` é no-op quando módulo já requerido naquela versão. Go informa caminhos certos na própria mensagem de erro.

Piso é `go 1.25.0`, não 1.24. Forçado por `go-sdk@v1.5.0`, que declara 1.25.0 no próprio go.mod.

## Regras que não são negociáveis

**stdout pertence ao JSON-RPC.** Todo log vai pra stderr via `log/slog`. `fmt.Println` em código alcançável de `serve` corrompe sessão — sintoma: servidor some do host sem erro nenhum. `doctor` e `version` imprimem em stdout de propósito — são comandos CLI, não servidores. Distinção merece comentário onde aparece.

**Nenhum pacote sob `internal/` ou `cmd/` importa `net` ou `net/*`.** `net/http` e `x/oauth2` chegam transitivamente pelo SDK — esperado. Check de CI inspeciona nossos pacotes, não fecho transitivo. Garantia verificável em PRD §6.4.

**Nenhum tipo do SDK MCP cruza pra fora de `internal/mcpsrv`.** `internal/service` fala tipos de domínio, não importa SDK. Torna migração de protocolo mudança de um pacote só — e protocolo já quebrou compat várias vezes.

**`ctx` onde há espera real.** Funções que podem **bloquear** recebem `ctx` e respeitam: I/O de arquivo, varredura, worker pool, watcher, chamadas MCP. Leitura de env var, resolução de caminho, cálculo em memória não recebem. `ctx` que nenhum corpo verifica ensina revisor a ignorar `ctx` — quando o parâmetro existe só por consistência de assinatura, nomeie-o `_`, como em `watchStdin`. **Não há mais exceção à regra.** `lifecycle.Shutdown` recebe `ctx` e descarta o cancelamento via `context.WithoutCancel`: o context raiz já está cancelado quando ela roda, então derivar os orçamentos dele faria toda etapa nascer expirada. `WithoutCancel` preserva os valores e joga fora só o cancelamento. Quem limita tempo ali são os orçamentos por etapa e o `hardLimit`.

**Código de plataforma atrás de build tag, em arquivo separado.** Nunca `if runtime.GOOS ==` dentro de lógica compartilhada. Em teste, `runtime.GOOS` aceitável pra pular casos.

**Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`. Console PowerShell em CP-850 renderiza resto como lixo, e `doctor` é justamente o comando que alguém roda quando já está confuso.

**Sem `helpers.go`, `utils.go`, `common.go`.** Arquivo assim é preocupação que ninguém nomeou.

Commits em Conventional Commits, em inglês.

## Armadilhas que já custaram caro

Cada uma passou por revisão e só apareceu depois. Estão aqui pra não voltarem.

**`io.TeeReader` não propaga EOF.** Copia bytes, EOF não é byte. Usar `mirrorReader`, que faz `dst.CloseWithError(err)`. Sem isso monitor de stdin do lifecycle fica inerte e `lc.Wait()` só retorna por acidente. Espelho é **auxiliar**: falha de escrita nele não pode virar erro da leitura principal — mataria sessão saudável por motivo que cliente não pode agir.

**Vigília do pai precisa de `exitTime`, não só creation time.** Windows mantém PID e creation time consultáveis muito tempo depois da morte do processo. Comparar `(pid, created)` nunca detecta pai morto — deixou 5 de 5 órfãos no primeiro teste ponta a ponta. Em Unix, comparar ppid capturado no startup, não constante 1: sob Docker+tini, systemd ou s6 o reaper não é PID 1.

**Goroutine parada em `Read` não é desenrolável por cancelamento de context.** Por isso `watchStdin` fica fora do `WaitGroup` — incluí-la trava `Wait()` quando sinal ou pai dispara primeiro. Vigias de sinal e de pai entram, porque fazem `select` em `ctx.Done()`.

**`ctx.Canceled` no retorno do serve loop é encerramento normal.** Duas detecções de EOF independentes correm — SDK e lifecycle — e qual vence decide o valor. Tratar como falha faz host ver erro aleatório a cada desconexão limpa.

**Handler que devolve `error` Go faz SDK montar `IsError` sem `StructuredContent`.** Devolver resultado de erro com `Out` zerado manda `{"notes":0,...}` junto, e cliente não distingue falha de cofre vazio no canal que ele lê primeiro.

**Confinamento de caminho tem duas camadas.** `validateLocal` (léxica: NUL, `..`, raiz, `IsLocal`, regra de plataforma) e `Canonicalize` (por componente, via `filepath.Rel`). `filepath.IsLocal` barra nome de dispositivo — `Resolve(root, "COM1")` escrevia em porta serial antes disso. Mas regra de ponto/espaço no fim de componente vale **só no Windows**: em Linux `Notas ` é nome legal, e rejeitá-lo lá torna notas reais inalcançáveis.

**`CanonicalPath` não garante grafia do disco.** Esta camada não consulta disco; preserva o que chamador passou. Quem produz grafia real é `vault.Walk`.

**`d == nil` no callback do `WalkDir` é falha na própria raiz** — cofre desmontado, share caído. Devolver `nil` ali faz varredura reportar sucesso com zero entradas, e servidor afirma com confiança que cofre está vazio. Cofre inacessível e cofre vazio não podem produzir mesma resposta.

**Flag booleana ou inteira não distingue "omitida" de "definida com zero".** `config.Flags` tem companheiros `ReadOnlySet` e `DebounceMSSet`. **Toda** chamada a `config.Load` precisa preenchê-los com `cmd.Flags().Changed(nome)` — esquecer em um subcomando faz flag virar no-op silencioso.

## Estado

M0 completa, etiquetada `m0-lifecycle`: ciclo de vida, `internal/vault`, servidor MCP mínimo com `vault_stats`, `doctor`, e 100 ciclos de encerramento abrupto com zero órfãos.

Dívida de revisão de M0 **paga**. Tasks 9, 10 e 11 haviam fechado sem revisão fresca; a revisão que faltava rodou e virou trabalho (plano `docs/superpowers/plans/2026-07-26-m0-review-fixes.md`). Três defeitos reais que os gates existentes não pegavam: `doctor` saindo 0 com cofre inacessível; o gate de órfãos não gateando em `reason=`, de modo que servidor morrendo sozinho dava rodada verde sem mecanismo nenhum disparar; e `cmd/gobsidian` sem teste algum.

Lint limpo nos três alvos (`GOOS=linux/darwin/windows`), depois de 39 achados que estavam vermelhos desde o commit de bootstrap. CI ganhou `fmt` (gofmt + vet cruzado) e `lint-windows`, sem o qual todo arquivo `//go:build windows` ficava sem análise.

Próximo é M1 (Task 12 em diante): parser, índice e as cinco tools de leitura, que fecham a v0.1. Task 25 precisa de uma rodada manual do plugin do Obsidian para gerar a referência de paridade — vale agendar antes de chegar nela.

Lacuna registrada pra M6: no harness de órfãos atual `stdin-eof` sempre vence (100/100 nas duas últimas rodadas), então vigília do pai e sinais seguem sem verificação ponta a ponta. Falta cenário em que stdin fica aberto e pai morre.