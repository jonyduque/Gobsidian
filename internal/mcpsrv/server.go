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

// Server e o adaptador entre o SDK de MCP e o servico de dominio. Ele detem
// o unico *mcp.Server do processo e e o unico ponto onde tipos do SDK
// aparecem.
type Server struct {
	mcp *mcp.Server
	svc *service.Service
	cfg config.Config
	log *slog.Logger
}

// New monta o servidor e registra as tools. Com cfg.ReadOnly ligado, as
// tools de escrita nao sao registradas — e nao apenas recusadas na chamada:
// um host que ve a tool anunciada vai tentar usa-la, e a recusa vira uma
// rodada desperdicada (PRD RF-55).
func New(svc *service.Service, cfg config.Config, log *slog.Logger) *Server {
	s := &Server{
		mcp: mcp.NewServer(&mcp.Implementation{Name: "gobsidian", Version: Version}, nil),
		svc: svc,
		cfg: cfg,
		log: log,
	}
	s.registerReadTools()
	s.registerReadToolsInternal()
	s.registerResources()
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
					return nil, service.StatsResult{}, toolErr(err)
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

// nopWriteCloser existe porque a biblioteca padrao tem io.NopCloser para
// leitura e nao tem o equivalente para escrita. Fechar stdout aqui seria
// errado: quem o abriu foi o processo, nao esta camada.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

// Serve liga o servidor a stdio atraves de IOTransport, e NAO de
// StdioTransport.
//
// A diferenca e o ponto todo: StdioTransport le os.Stdin diretamente, e o
// monitor de EOF do lifecycle tambem precisa ler stdin. Dois leitores no
// mesmo descritor repartem os bytes entre si e corrompem o JSON-RPC.
// IOTransport aceita o io.Reader que recebermos, o que permite passar um
// TeeReader: o transporte le, e a copia espelhada alimenta o lifecycle.
//
// mcp.IOTransport pede io.ReadCloser/io.WriteCloser (v1.5.0), nao
// io.Reader/io.Writer como o desenho original presumia. stdin e envolvido em
// io.NopCloser; stdout, em nopWriteCloser — nenhum dos dois adiciona
// comportamento de fechamento alem do nulo, e a assinatura publica de Serve
// continua em io.Reader/io.Writer para nao vazar esse detalhe do SDK.
func (s *Server) Serve(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	return s.mcp.Run(ctx, &mcp.IOTransport{
		Reader: io.NopCloser(stdin),
		Writer: nopWriteCloser{stdout},
	})
}

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
