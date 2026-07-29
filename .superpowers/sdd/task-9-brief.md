### Task 9: Servidor MCP mínimo com `vault_stats` trivial

**Files:**
- Create: `internal/service/errors.go`, `internal/service/service.go`
- Create: `internal/mcpsrv/server.go`, `internal/mcpsrv/convert.go`, `internal/mcpsrv/recover.go`
- Create: `internal/mcpsrv/server_test.go`
- Create: `cmd/gobsidian/main.go`, `cmd/gobsidian/serve.go`

**Interfaces:**
- Consumes: `config.Config` (Task 2), `lifecycle.New`/`Shutdown` (Tasks 3–6), `vault.Vault` (Task 8)
- Produces: `service.Service` com `VaultStats(ctx, StatsRequest) (StatsResult, error)`; `mcpsrv.New(svc *service.Service, cfg config.Config, log *slog.Logger) *mcpsrv.Server`; `(*Server).Serve(ctx context.Context, stdin io.Reader, stdout io.Writer) error`; erros sentinela de `service`

- [ ] **Step 1: Escrever o teste de handshake ponta a ponta**

`internal/mcpsrv/server_test.go`:

```go
package mcpsrv_test

import (
	"context"
	"log/slog"
	"io"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestServer(t *testing.T, root string) *mcpsrv.Server {
	t.Helper()

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	svc := service.New(v, nil, service.Options{})
	cfg := config.Defaults()

	return mcpsrv.New(svc, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestServerAnswersInitializeAndListsTools(t *testing.T) {
	srv := newTestServer(t, t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Transporte em memoria: os dois lados falam JSON-RPC sem tocar o disco
	// nem o processo. E o mesmo caminho de codigo do stdio.
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	found := false
	for _, tool := range tools.Tools {
		if tool.Name == "vault_stats" {
			found = true
		}
	}
	if !found {
		t.Fatal("vault_stats nao esta na lista de tools")
	}
}

func TestVaultStatsCountsNotes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n")
	writeFile(t, root, "sub/B.md", "# B\n")

	srv := newTestServer(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "vault_stats"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("vault_stats devolveu erro: %+v", res.Content)
	}
}

// Uma tool que entra em panic nao pode derrubar o servidor (RNF-13).
func TestPanicInHandlerBecomesToolError(t *testing.T) {
	srv := newTestServer(t, t.TempDir())
	srv.RegisterPanicProbeForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "panic_probe"})
	if err != nil {
		t.Fatalf("CallTool devolveu erro de transporte, quer erro de tool: %v", err)
	}
	if !res.IsError {
		t.Fatal("panic deveria virar resultado de erro de tool")
	}

	// O servidor continua respondendo depois do panic.
	if _, err := session.ListTools(ctx, nil); err != nil {
		t.Fatalf("servidor caiu apos panic em handler: %v", err)
	}
}
```

Adicione um `writeFile` local ao pacote de teste, idêntico ao de `internal/vault/walk_test.go`.

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/mcpsrv/ -v`
Esperado: FAIL — `undefined: mcpsrv.New`.

- [ ] **Step 3: Implementar a taxonomia de erros**

`internal/service/errors.go`:

```go
// Package service e a fachada unica sobre os subsistemas. Cada tool MCP
// corresponde a um metodo aqui. Nenhum tipo do SDK de MCP atravessa esta
// fronteira: o pacote fala Go de dominio, e a traducao acontece em mcpsrv.
package service

import "errors"

// Code e o codigo legivel por maquina devolvido ao cliente. A tabela completa
// esta em docs/TOOLS.md.
type Code string

const (
	CodePathOutsideVault Code = "PATH_OUTSIDE_VAULT"
	CodeNoteNotFound     Code = "NOTE_NOT_FOUND"
	CodeNoteExists       Code = "NOTE_ALREADY_EXISTS"
	CodeAmbiguousPath    Code = "AMBIGUOUS_PATH"
	CodeHeadingNotFound  Code = "HEADING_NOT_FOUND"
	CodeAmbiguousHeading Code = "AMBIGUOUS_HEADING"
	CodeBlockNotFound    Code = "BLOCK_NOT_FOUND"
	CodeHashMismatch     Code = "HASH_MISMATCH"
	CodeFileLocked       Code = "FILE_LOCKED"
	CodeCloudOnlyFile    Code = "CLOUD_ONLY_FILE"
	CodePathTooLong      Code = "PATH_TOO_LONG"
	CodeReadOnlyMode     Code = "READ_ONLY_MODE"
	CodeVaultUnavailable Code = "VAULT_UNAVAILABLE"
	CodeInternal         Code = "INTERNAL"
)

// Error carrega codigo e mensagem acionavel. A mensagem e lida por um modelo
// de linguagem que precisa decidir o que fazer em seguida: "heading nao
// encontrado" gera uma rodada extra de chamadas, enquanto a mesma mensagem
// listando os headings disponiveis permite que o cliente se corrija sozinho.
type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

func Errorf(code Code, format string, args ...any) *Error {
	return &Error{Code: code, Message: sprintf(format, args...)}
}

func Wrap(code Code, err error, format string, args ...any) *Error {
	return &Error{Code: code, Message: sprintf(format, args...), Err: err}
}

// CodeOf extrai o codigo de um erro, ou INTERNAL se ele nao carregar um.
func CodeOf(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeInternal
}
```

Adicione `sprintf` como alias local de `fmt.Sprintf` no mesmo arquivo, importando `fmt`.

- [ ] **Step 4: Implementar a fachada `Service` com `VaultStats`**

`internal/service/service.go`:

```go
package service

import (
	"context"

	"github.com/jonyd/gobsidian/internal/vault"
)

// Index e a dependencia do servico sobre o indice, declarada como interface
// para que o servico seja testavel sem construir um indice completo, e para
// que M1 possa injetar a implementacao real sem tocar aqui.
type Index interface {
	NoteCount() int
	AssetCount() int
	TotalSize() int64
}

type Options struct {
	ReadOnly   bool
	MaxResults int
}

type Service struct {
	vault *vault.Vault
	index Index
	opts  Options
}

func New(v *vault.Vault, idx Index, opts Options) *Service {
	return &Service{vault: v, index: idx, opts: opts}
}

type StatsRequest struct {
	IncludeHealth  bool `json:"include_health"`
	IncludeRuntime bool `json:"include_runtime"`
}

type StatsResult struct {
	Notes     int   `json:"notes"`
	Assets    int   `json:"assets"`
	TotalSize int64 `json:"total_size"`
}

// VaultStats em M0 conta arquivos varrendo o disco. Em M1 passa a ler do
// indice; a assinatura nao muda, e o teste desta tarefa continua valendo.
func (s *Service) VaultStats(ctx context.Context, req StatsRequest) (StatsResult, error) {
	if s.index != nil {
		return StatsResult{
			Notes:     s.index.NoteCount(),
			Assets:    s.index.AssetCount(),
			TotalSize: s.index.TotalSize(),
		}, nil
	}

	var out StatsResult
	err := s.vault.Walk(ctx, func(e vault.Entry) error {
		if e.IsNote {
			out.Notes++
		} else {
			out.Assets++
		}
		out.TotalSize += e.Size
		return nil
	})
	if err != nil {
		return StatsResult{}, Wrap(CodeVaultUnavailable, err, "varrendo o cofre: %v", err)
	}
	return out, nil
}
```

- [ ] **Step 5: Implementar a camada MCP**

`internal/mcpsrv/recover.go`:

```go
package mcpsrv

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// guard embrulha um handler de tool de modo que um panic vire resultado de
// erro, com stack trace em stderr. RNF-13: falha de uma tool jamais derruba
// o servidor. E o que distingue um servidor robusto de um que exige reiniciar
// o Claude Desktop toda vez que um caminho invalido e passado.
func guard[In, Out any](
	log *slog.Logger,
	name string,
	fn func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error),
) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (res *mcp.CallToolResult, out Out, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic em handler de tool",
					"tool", name,
					"panic", fmt.Sprint(r),
					"stack", string(debug.Stack()))
				res = errorResult(string(codeInternal), "falha interna em "+name+"; detalhes registrados em stderr")
				err = nil
			}
		}()
		return fn(ctx, req, in)
	}
}
```

`internal/mcpsrv/convert.go`:

```go
package mcpsrv

import (
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const codeInternal = service.CodeInternal

// errorResult monta o resultado de erro no formato que o host entende, com
// codigo legivel por maquina no inicio da mensagem.
func errorResult(code, message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: code + ": " + message},
		},
	}
}

// toolError traduz um erro de dominio em resultado MCP. Erros nunca sobem
// como erro de protocolo: o cliente precisa poder ler a mensagem e se corrigir.
func toolError(err error) *mcp.CallToolResult {
	return errorResult(string(service.CodeOf(err)), err.Error())
}
```

`internal/mcpsrv/server.go`:

```go
// Package mcpsrv adapta o SDK oficial de MCP ao dominio do produto.
//
// Esta camada existe para isolar a instabilidade do SDK. O protocolo evoluiu
// com quebras entre versoes, e concentrar o contato com o SDK em um pacote
// significa que uma quebra de API se resolve aqui, nao espalhada pelo codigo.
// Nenhum tipo do SDK atravessa a fronteira para service ou abaixo.
package mcpsrv

import (
	"context"
	"io"
	"log/slog"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version e injetada pelo linker no build; o valor aqui e o fallback de
// desenvolvimento.
var Version = "dev"

type Server struct {
	mcp *mcp.Server
	svc *service.Service
	cfg config.Config
	log *slog.Logger
}

func New(svc *service.Service, cfg config.Config, log *slog.Logger) *Server {
	s := &Server{
		mcp: mcp.NewServer(&mcp.Implementation{Name: "gobsidian", Version: Version}, nil),
		svc: svc,
		cfg: cfg,
		log: log,
	}
	s.registerReadTools()
	if !cfg.ReadOnly {
		s.registerWriteTools()
	}
	return s
}

type statsInput struct {
	IncludeHealth  bool `json:"include_health,omitempty" jsonschema:"inclui contagem de orfas, links quebrados e ancoras quebradas"`
	IncludeRuntime bool `json:"include_runtime,omitempty" jsonschema:"inclui RSS, goroutines e contadores do watcher"`
}

func (s *Server) registerReadTools() {
	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "vault_stats",
			Description: "Estado do cofre e saude do servidor: contagens, orfas, links quebrados, anexos.",
		},
		guard(s.log, "vault_stats",
			func(ctx context.Context, _ *mcp.CallToolRequest, in statsInput) (*mcp.CallToolResult, service.StatsResult, error) {
				out, err := s.svc.VaultStats(ctx, service.StatsRequest{
					IncludeHealth:  in.IncludeHealth,
					IncludeRuntime: in.IncludeRuntime,
				})
				if err != nil {
					return toolError(err), service.StatsResult{}, nil
				}
				return nil, out, nil
			}),
	)
}

// registerWriteTools fica vazia ate M4. A separacao existe desde ja porque
// --read-only precisa REMOVER as tools da lista anunciada, nao apenas
// rejeitar a chamada (PRD RF-55): um host que ve a tool na lista vai tentar
// usa-la, e a recusa vira uma rodada desperdicada.
func (s *Server) registerWriteTools() {}

// Connect liga o servidor a um transporte ja construido. Usado nos testes com
// transporte em memoria.
func (s *Server) Connect(ctx context.Context, t mcp.Transport) error {
	return s.mcp.Run(ctx, t)
}

// Serve liga o servidor a stdio atraves de IOTransport, e NAO de
// StdioTransport.
//
// A diferenca e o ponto todo: StdioTransport le os.Stdin diretamente, e o
// monitor de EOF do lifecycle tambem precisa ler stdin. Dois leitores no
// mesmo descritor repartem os bytes entre si e corrompem o JSON-RPC.
// IOTransport aceita o io.Reader que recebermos, o que permite passar um
// TeeReader: o transporte le, e a copia espelhada alimenta o lifecycle.
func (s *Server) Serve(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	return s.mcp.Run(ctx, &mcp.IOTransport{Reader: stdin, Writer: stdout})
}
```

Adicione ao final de `server.go`, com build tag de teste ou como método normal marcado no nome:

```go
// RegisterPanicProbeForTest registra uma tool que sempre entra em panic.
// Existe para provar que RNF-13 vale — nao e registrada em producao.
func (s *Server) RegisterPanicProbeForTest() {
	type empty struct{}
	mcp.AddTool(s.mcp,
		&mcp.Tool{Name: "panic_probe", Description: "sonda de teste; entra em panic"},
		guard(s.log, "panic_probe",
			func(context.Context, *mcp.CallToolRequest, empty) (*mcp.CallToolResult, empty, error) {
				panic("sonda")
			}),
	)
}
```

- [ ] **Step 6: Rodar para confirmar que passa**

Run: `go test -race ./internal/mcpsrv/ -v`
Esperado: PASS, três testes.

Se `mcp.NewInMemoryTransports` não existir com esse nome em `v1.5.0`, consulte `go doc github.com/modelcontextprotocol/go-sdk/mcp | grep -i transport` e use o construtor equivalente. **Não** substitua por um teste que apenas chama métodos internos: o valor deste teste é atravessar o JSON-RPC de verdade.

- [ ] **Step 7: Implementar a CLI**

`cmd/gobsidian/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Injetados pelo linker. Ver scripts/build.ps1.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:           "gobsidian",
		Short:         "Servidor MCP para cofres locais do Obsidian",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newServeCmd(), newDoctorCmd(), newVersionCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "[!] %v\n", err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Imprime versao, commit e data de build",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "gobsidian %s (%s) %s\n", version, commit, buildDate)
		},
	}
}
```

`cmd/gobsidian/serve.go`:

```go
package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/lifecycle"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var flags config.Flags

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve o cofre via MCP sobre stdio",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Flags booleanas e inteiras nao distinguem "omitida" de "definida
			// com o valor zero". Sem isso, --read-only=false nao consegue
			// sobrepor GOBSIDIAN_READ_ONLY=true, e --debounce-ms=0 e
			// indistinguivel de nao passar a flag.
			flags.ReadOnlySet = cmd.Flags().Changed("read-only")
			flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}
			return runServe(cmd.Context(), cfg)
		},
	}

	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().StringVar(&flags.LogLevel, "log-level", "", "debug, info, warn ou error")
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "desabilita toda a superficie de escrita")
	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")
	cmd.Flags().StringVar(&flags.CacheDir, "cache-dir", "", "diretorio do cache de indice")

	return cmd
}

func runServe(parent context.Context, cfg config.Config) error {
	// stderr, sempre. stdout carrega o JSON-RPC e um unico byte estranho
	// corrompe a sessao.
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel}))

	v, err := vault.New(cfg.VaultPath)
	if err != nil {
		return err
	}

	// O monitor de stdin consome bytes, e o stdin aqui pertence ao JSON-RPC.
	// A saida e espelhar: o SDK le do TeeReader, e o lifecycle observa a copia.
	// Quando o host fecha o stdin, ambos veem EOF.
	pr, pw := io.Pipe()
	teed := io.TeeReader(os.Stdin, pw)

	ctx, lc := lifecycle.New(parent, lifecycle.Options{
		Stdin:     pr,
		ParentPID: lifecycle.ParentPID(),
		Logger:    log,
	})

	svc := service.New(v, nil, service.Options{
		ReadOnly:   cfg.ReadOnly,
		MaxResults: cfg.MaxResults,
	})
	srv := mcpsrv.New(svc, cfg, log)

	log.Info("servidor pronto", "vault", cfg.VaultPath, "read_only", cfg.ReadOnly)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ctx, teed, os.Stdout) }()

	select {
	case err := <-serveErr:
		if err != nil {
			log.Error("servidor encerrou com erro", "err", err)
		}
	case <-ctx.Done():
	}

	lifecycle.Shutdown(log, 6*time.Second,
		lifecycle.Step{Name: "in-flight", Budget: 3 * time.Second, Fn: func(ctx context.Context) error {
			select {
			case <-serveErr:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		}},
		lifecycle.Step{Name: "close-pipe", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
			return pw.Close()
		}},
	)

	lc.Wait()
	os.Exit(0)
	return nil
}
```

**Nota sobre o `TeeReader`.** Ele resolve o conflito de leitor único, e tem um custo: cada byte do JSON-RPC é copiado para o pipe. Se o lifecycle não drenar, o `TeeReader` bloqueia e o servidor trava. O monitor de stdin drena em laço e descarta, que é exatamente o comportamento necessário. Verifique isso ao rodar o Step 8: se o servidor congelar após a primeira mensagem, é aqui.

**Confirme a forma exata de `IOTransport` antes de escrever o código.** O SDK expõe `IOTransport`, mas os nomes dos campos precisam ser verificados na versão fixada:

```powershell
go doc github.com/modelcontextprotocol/go-sdk/mcp.IOTransport
```

Se os campos não forem `Reader` e `Writer`, ajuste a chamada — **não** volte para `StdioTransport`, que reintroduz o conflito de leitor. Se `IOTransport` não existir nesta versão, pare e reporte: o desenho do lifecycle em `serve` depende dele.

- [ ] **Step 8: Verificar o handshake manualmente**

```powershell
go build -o bin\gobsidian.exe .\cmd\gobsidian
$Req = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"manual","version":"1.0"}}}'
$Req | .\bin\gobsidian.exe serve --vault "C:\caminho\do\cofre" 2> $null
```

Esperado: **uma única linha JSON em stdout**, com `"result"` e `"protocolVersion"`. Qualquer outra coisa em stdout — banner, log, aviso — é a causa de o servidor não aparecer no host, e precisa ser corrigida antes de seguir.

- [ ] **Step 9: Commit**

```bash
git add internal/service internal/mcpsrv cmd/gobsidian
git commit -m "feat(mcpsrv): minimal stdio server with vault_stats and panic recovery"
```

---

