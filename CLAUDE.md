# gobsidian

Servidor MCP em Go que expõe um cofre local do Obsidian a hosts MCP, rodando como subprocesso sobre stdio.

A especificação vive em `docs/`: `PRD.md` (requisitos e decisões fechadas D1–D13), `ARCHITECTURE.md` (camadas e decisões AD-01–AD-09), `ESTRUTURA.md` (árvore e convenções), `TOOLS.md` (contrato de cada tool), `WINDOWS.md` (OneDrive, MAX_PATH, casing). O plano de implementação está em `docs/superpowers/plans/2026-07-25-gobsidian-v01.md` e é a fonte de onde o código é transcrito — plano e código não podem divergir.

## Comandos

```bash
go test -race ./...          # a versão sem -race não conta
go vet ./...
gofmt -l .
GOOS=linux go vet ./...      # e darwin: há arquivos com build tag
pwsh -File scripts/check_net.ps1
pwsh -File scripts/build.ps1
pwsh -File scripts/test_orphans.ps1 -Cycles 100
```

## Dependências

**Nunca rode `go mod tidy`.** Várias dependências estão fixadas mas ainda não têm importador — `goldmark`, `yaml.v3`, `x/text` só ganham um em M1 e M3. `tidy` as removeria, junto com o pin do SDK de MCP, que é decisão fechada (PRD D6).

Quando um pacote passa a importar uma dependência pela primeira vez, o `go.sum` pode não ter as entradas transitivas. O comando correto é `go get <caminho-do-pacote>@<versão-fixada>` — o caminho do **pacote**, não do módulo. `go get <módulo>@<versão>` é no-op quando o módulo já está requerido naquela versão, e o Go informa os caminhos certos na própria mensagem de erro.

O piso é `go 1.25.0`, não 1.24. Foi forçado por `go-sdk@v1.5.0`, que declara 1.25.0 no próprio go.mod.

## Regras que não são negociáveis

**stdout pertence ao JSON-RPC.** Todo log vai para stderr via `log/slog`. Um `fmt.Println` em código alcançável a partir de `serve` corrompe a sessão, e o sintoma é o servidor sumir do host sem erro nenhum. `doctor` e `version` imprimem em stdout de propósito — são comandos de CLI, não servidores, e a distinção merece comentário onde aparece.

**Nenhum pacote sob `internal/` ou `cmd/` importa `net` ou `net/*`.** `net/http` e `x/oauth2` chegam transitivamente pelo SDK e isso é esperado — o check de CI inspeciona os nossos pacotes, não o fecho transitivo. A garantia verificável está em PRD §6.4.

**Nenhum tipo do SDK de MCP cruza para fora de `internal/mcpsrv`.** `internal/service` fala tipos de domínio e não importa o SDK. É o que torna uma migração de protocolo uma mudança de um pacote só — e o protocolo já quebrou compatibilidade várias vezes.

**`ctx` onde há espera real.** Funções que podem **bloquear** recebem `ctx` e o respeitam: I/O de arquivo, varredura, worker pool, watcher, chamadas MCP. Leitura de variável de ambiente, resolução de caminho e cálculo em memória não recebem. Um `ctx` que nenhum corpo verifica ensina o revisor a ignorar `ctx`. Exceção única e documentada: `lifecycle.Shutdown` bloqueia e não recebe, porque o context raiz já está cancelado quando ela roda.

**Código de plataforma atrás de build tag, em arquivo separado.** Nunca `if runtime.GOOS ==` dentro de lógica compartilhada. Em teste, `runtime.GOOS` é aceitável para pular casos.

**Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`. Console PowerShell em CP-850 renderiza o resto como lixo, e `doctor` é justamente o comando que alguém roda quando já está confuso.

**Sem `helpers.go`, `utils.go`, `common.go`.** Arquivo assim é preocupação que ninguém nomeou.

Commits em Conventional Commits, em inglês.

## Armadilhas que já custaram caro

Cada uma abaixo passou por revisão e só apareceu depois. Estão aqui para não voltarem.

**`io.TeeReader` não propaga EOF.** Copia bytes, e EOF não é byte. Usar `mirrorReader`, que faz `dst.CloseWithError(err)`. Sem isso o monitor de stdin do lifecycle fica inerte e `lc.Wait()` só retorna por acidente. O espelho é **auxiliar**: falha de escrita nele não pode virar erro da leitura principal, senão mata sessão saudável por motivo que o cliente não pode agir.

**Vigília do pai precisa de `exitTime`, não só creation time.** O Windows mantém PID e creation time consultáveis por muito tempo depois da morte do processo. Comparar `(pid, created)` nunca detecta o pai morto — deixou 5 de 5 órfãos no primeiro teste ponta a ponta. Em Unix, comparar o ppid capturado no startup, não a constante 1: sob Docker+tini, systemd ou s6 o reaper não é o PID 1.

**Uma goroutine parada em `Read` não é desenrolável por cancelamento de context.** Por isso `watchStdin` fica fora do `WaitGroup` — incluí-la trava `Wait()` quando sinal ou pai dispara primeiro. Os vigias de sinal e de pai entram, porque fazem `select` em `ctx.Done()`.

**`ctx.Canceled` no retorno do serve loop é encerramento normal.** Duas detecções de EOF independentes correm — a do SDK e a do lifecycle — e qual vence decide o valor. Tratar como falha faz o host ver erro aleatório a cada desconexão limpa.

**Handler que devolve `error` Go faz o SDK montar `IsError` sem `StructuredContent`.** Devolver resultado de erro com o `Out` zerado manda `{"notes":0,...}` junto, e o cliente não distingue falha de cofre vazio no canal que ele lê primeiro.

**Confinamento de caminho tem duas camadas.** `validateLocal` (léxica: NUL, `..`, raiz, `IsLocal`, regra de plataforma) e `Canonicalize` (por componente, via `filepath.Rel`). `filepath.IsLocal` é o que barra nome de dispositivo — `Resolve(root, "COM1")` escrevia em porta serial antes disso. Mas a regra de ponto/espaço no fim de componente vale **só no Windows**: em Linux `Notas ` é nome legal, e rejeitá-lo lá torna notas reais inalcançáveis.

**`CanonicalPath` não garante a grafia do disco.** Esta camada não consulta disco; preserva o que o chamador passou. Quem produz grafia real é `vault.Walk`.

**`d == nil` no callback do `WalkDir` é falha na própria raiz** — cofre desmontado, share caído. Devolver `nil` ali faz a varredura reportar sucesso com zero entradas, e o servidor afirma com confiança que o cofre está vazio. Cofre inacessível e cofre vazio não podem produzir a mesma resposta.

**Flag booleana ou inteira não distingue "omitida" de "definida com zero".** `config.Flags` tem os companheiros `ReadOnlySet` e `DebounceMSSet`. **Toda** chamada a `config.Load` precisa preenchê-los com `cmd.Flags().Changed(nome)` — esquecer em um subcomando faz a flag virar no-op silencioso.

## Estado

M0 completa e etiquetada `m0-lifecycle`: ciclo de vida, `internal/vault`, servidor MCP mínimo com `vault_stats`, `doctor`, e 100 ciclos de encerramento abrupto com zero órfãos.

Próximo é M1 (Task 12 em diante): parser, índice e as cinco tools de leitura, que fecham a v0.1.

Lacuna registrada para M6: no harness de órfãos atual o `stdin-eof` sempre vence, então vigília do pai e sinais seguem sem verificação ponta a ponta. Falta um cenário em que o stdin fica aberto e o pai morre.
