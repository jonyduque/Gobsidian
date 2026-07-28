package mcpsrv

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerResources() {
	// Handler compartilhado
	handler := func(ctx context.Context, req *mcp.ReadResourceRequest) (res *mcp.ReadResourceResult, err error) {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("panic em handler de resource",
					"uri", req.Params.URI,
					"panic", fmt.Sprint(r),
					"stack", string(debug.Stack()))
				res = nil
				err = fmt.Errorf("falha interna no resource; detalhes registrados em stderr")
			}
		}()

		uri := req.Params.URI
		if !strings.HasPrefix(uri, "gobsidian://") {
			return nil, fmt.Errorf("invalid resource URI schema: %s", uri)
		}
		path := strings.TrimPrefix(uri, "gobsidian://")

		noteRes, err := s.svc.ReadNote(ctx, service.ReadRequest{
			Path:               path,
			IncludeFrontmatter: true,
		})
		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "text/markdown",
					Text:     noteRes.Content,
				},
			},
		}, nil
	}

	// Adiciona template genérico para permitir leitura de qualquer nota (mesmo se não listada)
	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "gobsidian://{path}",
		Name:        "Nota do cofre",
		MIMEType:    "text/markdown",
	}, handler)

	// Busca as 200 últimas notas modificadas
	ctx := context.Background()
	res, err := s.svc.ListNotes(ctx, service.ListRequest{
		Query: index.Query{Limit: 200, Sort: "modified", Order: "desc"},
	})
	if err != nil {
		s.log.Error("falha ao listar notas para resources", "err", err)
		return
	}

	for _, n := range res.Notes {
		name := n.Title
		if name == "" {
			name = string(n.Path)
		}
		s.mcp.AddResource(&mcp.Resource{
			URI:      "gobsidian://" + string(n.Path),
			Name:     name,
			MIMEType: "text/markdown",
		}, handler)
	}
}
