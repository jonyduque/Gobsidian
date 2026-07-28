package mcpsrv

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerResources(ctx context.Context) {
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
		path, err := pathFromResourceURI(uri)
		if err != nil {
			return nil, err
		}

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

	// Template generico: permite ler qualquer nota, inclusive as que ficaram
	// fora das 200 listadas abaixo.
	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "gobsidian:///{+path}",
		Name:        "Nota do cofre",
		MIMEType:    "text/markdown",
	}, handler)

	// Busca as 200 ultimas notas modificadas. O ctx vem de quem construiu o
	// servidor: usar context.Background() aqui desligaria este trecho do
	// cancelamento do processo, e um cofre grande faz dele um trecho que
	// demora.
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
			URI:      resourceURI(vault.CanonicalPath(n.Path)),
			Name:     name,
			MIMEType: "text/markdown",
		}, handler)
	}
}
