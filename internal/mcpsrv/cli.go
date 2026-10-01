package mcpsrv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EsquemaDeTool descreve uma tool em tipos do dominio: o que a CLI precisa
// para montar um comando -- nome, descricao e parametros -- sem conhecer o SDK.
type EsquemaDeTool struct {
	Nome       string
	Descricao  string
	Parametros []Parametro
	// Escrita diz que a tool so existe fora do modo somente leitura.
	Escrita bool
}

// Parametro e uma propriedade do schema de entrada de uma tool.
type Parametro struct {
	Nome        string
	Descricao   string
	Tipo        string // string, integer, number, boolean, array, object
	TipoDoItem  string // o tipo do item, quando Tipo e array
	Enum        []string
	Obrigatorio bool
	// ItemAceitaObjeto diz que o item do array aceita tambem um objeto (o
	// oneOf de note_read.paths). A CLI passa so a forma string; a forma objeto
	// vai por --args.
	ItemAceitaObjeto bool
}

// ErroDeTool e a resposta de erro de uma tool: o codigo de dominio
// (NOTE_NOT_FOUND, INVALID_ARGUMENT...) e a mensagem.
type ErroDeTool struct {
	Codigo   string
	Mensagem string
}

func (e *ErroDeTool) Error() string { return e.Codigo + ": " + e.Mensagem }

var (
	esquemasUmaVez sync.Once
	esquemas       []EsquemaDeTool
	errEsquemas    error
)

// EsquemasDeEntrada devolve o schema de cada tool, lido do servidor como um
// cliente o le -- pelo tools/list, e nao pelos tipos Go de entrada. E o mesmo
// schema que o host recebe; ler os tipos diretamente seria uma segunda conta
// que divergiria no primeiro remendo a mao, como o oneOf de note_read.paths.
//
// Monta o servidor sem servico nenhum: registrar uma tool nao chama o
// servico, e listar tambem nao. O resultado e calculado uma vez por processo.
func EsquemasDeEntrada() ([]EsquemaDeTool, error) {
	esquemasUmaVez.Do(func() {
		esquemas, errEsquemas = lerEsquemas()
	})
	return esquemas, errEsquemas
}

func lerEsquemas() ([]EsquemaDeTool, error) {
	ctx := context.Background()
	cfg := config.Defaults()
	todas, err := listarTools(ctx, cfg)
	if err != nil {
		return nil, err
	}
	cfg.ReadOnly = true
	soLeitura, err := listarTools(ctx, cfg)
	if err != nil {
		return nil, err
	}
	leitura := map[string]bool{}
	for _, t := range soLeitura {
		leitura[t.Name] = true
	}

	saida := make([]EsquemaDeTool, 0, len(todas))
	for _, t := range todas {
		params, err := parametrosDoSchema(t.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("schema de %s: %w", t.Name, err)
		}
		saida = append(saida, EsquemaDeTool{
			Nome:       t.Name,
			Descricao:  t.Description,
			Parametros: params,
			Escrita:    !leitura[t.Name],
		})
	}
	sort.Slice(saida, func(i, j int) bool { return saida[i].Nome < saida[j].Nome })
	return saida, nil
}

func listarTools(ctx context.Context, cfg config.Config) ([]*mcp.Tool, error) {
	s := novo(ctx, nil, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), false)
	sessao, parar, err := conectarLocal(ctx, s)
	if err != nil {
		return nil, err
	}
	defer parar()

	var tools []*mcp.Tool
	for t, err := range sessao.Tools(ctx, nil) {
		if err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}
	return tools, nil
}

// conectarLocal liga um cliente ao servidor por transporte em memoria. parar
// fecha a sessao e espera o servidor sair.
func conectarLocal(ctx context.Context, s *Server) (*mcp.ClientSession, func(), error) {
	ctx, cancel := context.WithCancel(ctx)
	clienteT, servidorT := mcp.NewInMemoryTransports()
	fim := make(chan struct{})
	go func() {
		defer close(fim)
		_ = s.Connect(ctx, servidorT)
	}()
	cliente := mcp.NewClient(&mcp.Implementation{Name: "gobsidian-cli", Version: Version}, nil)
	sessao, err := cliente.Connect(ctx, clienteT, nil)
	if err != nil {
		cancel()
		<-fim
		return nil, nil, err
	}
	parar := func() {
		_ = sessao.Close()
		cancel()
		<-fim
	}
	return sessao, parar, nil
}

// esquemaJSON e o pedaco do JSON Schema que a CLI le. Decodificar por JSON,
// e nao pelo tipo do SDK, e o que mantem a forma do SDK fora do dominio.
type esquemaJSON struct {
	Type        json.RawMessage        `json:"type"`
	Description string                 `json:"description"`
	Enum        []any                  `json:"enum"`
	Items       *esquemaJSON           `json:"items"`
	OneOf       []esquemaJSON          `json:"oneOf"`
	Properties  map[string]esquemaJSON `json:"properties"`
	Required    []string               `json:"required"`
}

// tipo devolve o tipo principal: "type" pode ser uma string ou uma lista com
// "null" -- e o que a inferencia produz para um campo ponteiro.
func (e esquemaJSON) tipo() string {
	if len(e.Type) == 0 {
		return ""
	}
	var um string
	if json.Unmarshal(e.Type, &um) == nil {
		return um
	}
	var varios []string
	if json.Unmarshal(e.Type, &varios) == nil {
		for _, t := range varios {
			if t != "null" {
				return t
			}
		}
	}
	return ""
}

func parametrosDoSchema(schema any) ([]Parametro, error) {
	b, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var e esquemaJSON
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, err
	}
	obrigatorio := map[string]bool{}
	for _, r := range e.Required {
		obrigatorio[r] = true
	}
	nomes := make([]string, 0, len(e.Properties))
	for n := range e.Properties {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)

	params := make([]Parametro, 0, len(nomes))
	for _, n := range nomes {
		p := e.Properties[n]
		par := Parametro{
			Nome:        n,
			Descricao:   p.Description,
			Tipo:        p.tipo(),
			Obrigatorio: obrigatorio[n],
		}
		for _, v := range p.Enum {
			par.Enum = append(par.Enum, fmt.Sprint(v))
		}
		if p.Items != nil {
			par.TipoDoItem = p.Items.tipo()
			for _, o := range p.Items.OneOf {
				switch o.tipo() {
				case "string":
					par.TipoDoItem = "string"
				case "object":
					par.ItemAceitaObjeto = true
				}
			}
			if par.Descricao == "" {
				par.Descricao = p.Items.Description
			}
		}
		params = append(params, par)
	}
	return params, nil
}

// ChamarLocal chama uma tool sobre o servico dado, pelo mesmo caminho que um
// host usa -- o servidor inteiro, com a validacao de schema do SDK e os
// padroes de cada tool --, por transporte em memoria.
//
// Devolve o structuredContent da tool em JSON, ou *ErroDeTool com o codigo de
// dominio. O servidor montado aqui nao publica resources: publicar percorre o
// cofre inteiro, e a CLI nao le nenhum.
func ChamarLocal(ctx context.Context, svc *service.Service, cfg config.Config, log *slog.Logger, nome string, argumentos map[string]any) (json.RawMessage, error) {
	s := novo(ctx, svc, cfg, log, false)
	sessao, parar, err := conectarLocal(ctx, s)
	if err != nil {
		return nil, err
	}
	defer parar()

	res, err := sessao.CallTool(ctx, &mcp.CallToolParams{Name: nome, Arguments: argumentos})
	if err != nil {
		// Erro de protocolo: a tool nao existe, ou a entrada nao passou pela
		// validacao de schema do SDK. Para quem chamou, e argumento invalido.
		return nil, &ErroDeTool{Codigo: string(service.CodeInvalidArgument), Mensagem: err.Error()}
	}
	if res.IsError {
		return nil, erroDoResultado(res)
	}
	return json.Marshal(res.StructuredContent)
}

// erroDoResultado le o erro que toolErr escreveu: "CODIGO: mensagem".
func erroDoResultado(res *mcp.CallToolResult) error {
	var texto string
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			texto = t.Text
			break
		}
	}
	if codigo, msg, ok := strings.Cut(texto, ": "); ok && codigo == strings.ToUpper(codigo) && !strings.Contains(codigo, " ") {
		return &ErroDeTool{Codigo: codigo, Mensagem: msg}
	}
	if texto == "" {
		return errors.New("a tool devolveu erro sem mensagem")
	}
	return &ErroDeTool{Codigo: string(codeInternal), Mensagem: texto}
}
