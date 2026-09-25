package mcpsrv

import (
	"context"
	"fmt"
	"path"
	"runtime/debug"
	"strings"

	"github.com/jonyduque/Gobsidian/internal/service"
	"github.com/jonyduque/Gobsidian/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	mimeNota  = "text/markdown"
	mimePasta = "inode/directory"

	// separadorDeContexto separa a folha do caminho da pasta no Title.
	separadorDeContexto = " · "
	// prefixoDePasta distingue pasta de nota pelo texto, porque o Claude
	// Desktop 1.52386.6 nao desenha icone de resource (medido em 2026-09-14,
	// E2). E o que o exemplo da propria spec faz: "📁 Project Files".
	prefixoDePasta = "📁 "
	tituloDaRaiz   = prefixoDePasta + "Cofre inteiro"
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
		p, err := pathFromResourceURI(uri)
		if err != nil {
			return nil, err
		}

		if ehPastaURI(uri) {
			return s.lerPasta(ctx, uri, p)
		}

		noteRes, err := s.svc.ReadNote(ctx, service.ReadRequest{
			Path:               p,
			IncludeFrontmatter: true,
		})
		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: mimeNota,
					Text:     noteRes.Content,
				},
			},
		}, nil
	}

	// Template generico: atende a URI que o cliente monta sozinho e que nao
	// esta na lista abaixo — outra caixa, a forma antiga de duas barras, ou
	// uma nota criada depois do boot.
	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "gobsidian:///{+path}",
		Name:        "Nota do cofre",
		MIMEType:    mimeNota,
	}, handler)

	// O cofre inteiro, ordenado por caminho, com as pastas. Ate 2026-09-25
	// eram as 200 notas mais recentes, e no cofre Estudo 94% das notas nunca
	// apareciam; a paginacao de resources/list e do SDK (1.000 por pagina) e o
	// Claude Desktop segue o cursor (E1). O ctx vem de quem construiu o
	// servidor: usar context.Background() aqui desligaria este trecho do
	// cancelamento do processo.
	arvore, err := s.svc.VaultTree(ctx, service.TreeRequest{Recursive: true})
	if err != nil {
		s.log.Error("falha ao montar a arvore do cofre para resources", "err", err)
		return
	}

	s.mcp.AddResource(recursoDePasta(arvore.Folder), handler)
	for _, f := range arvore.Folders {
		s.mcp.AddResource(recursoDePasta(f), handler)
	}
	for _, n := range arvore.Notes {
		s.mcp.AddResource(recursoDeNota(n), handler)
	}
}

// recursoDeNota: Name carrega o caminho sem extensao, que e por onde a busca
// do cliente casa; Title carrega a folha primeiro e a pasta depois, porque o
// Claude Desktop mostra so o Title, num menu estreito que corta pelo fim
// (E4) — cem "Nota 001" de pastas diferentes precisam da pasta visivel.
func recursoDeNota(n service.TreeNote) *mcp.Resource {
	return &mcp.Resource{
		URI:      resourceURI(vault.CanonicalPath(n.Path)),
		Name:     strings.TrimSuffix(n.Path, path.Ext(n.Path)),
		Title:    comContexto(folhaDaNota(n), n.Folder),
		MIMEType: mimeNota,
	}
}

// recursoDePasta publica a pasta com barra final na URI e no Name. A raiz e o
// item "o cofre inteiro".
func recursoDePasta(f service.FolderItem) *mcp.Resource {
	if f.Path == "" {
		return &mcp.Resource{
			URI:      resourceURI(""),
			Name:     "/",
			Title:    tituloDaRaiz,
			MIMEType: mimePasta,
		}
	}
	return &mcp.Resource{
		URI:      uriDePasta(f.Path),
		Name:     f.Path + "/",
		Title:    prefixoDePasta + comContexto(path.Base(f.Path), f.Parent),
		MIMEType: mimePasta,
	}
}

// folhaDaNota e o rotulo da nota: o titulo, ou o nome do arquivo sem extensao
// quando o titulo e vazio — nunca o caminho inteiro, que o Name ja carrega.
func folhaDaNota(n service.TreeNote) string {
	if n.Title != "" {
		return n.Title
	}
	return strings.TrimSuffix(path.Base(n.Path), path.Ext(n.Path))
}

func comContexto(folha, pasta string) string {
	if pasta == "" {
		return folha
	}
	return folha + separadorDeContexto + pasta
}

// lerPasta devolve UM conteudo: um indice em Markdown dos filhos diretos,
// cada um como link gobsidian:///. A spec permite devolver o conteudo de
// varios arquivos no read de um diretorio, mas a raiz de um cofre de estudo
// seriam milhares de notas numa resposta.
func (s *Server) lerPasta(ctx context.Context, uri, p string) (*mcp.ReadResourceResult, error) {
	arvore, err := s.svc.VaultTree(ctx, service.TreeRequest{Folder: p})
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	f := arvore.Folder
	if f.Path == "" {
		b.WriteString("# " + tituloDaRaiz + "\n\n")
	} else {
		b.WriteString("# " + prefixoDePasta + f.Path + "/\n\n")
	}
	fmt.Fprintf(&b, "%s no total, %s diretamente nesta pasta.\n", contarNotas(f.NotesRecursive), contarNotas(f.Notes))

	if len(arvore.Folders) > 0 {
		b.WriteString("\n## Subpastas\n\n")
		for _, sub := range arvore.Folders {
			fmt.Fprintf(&b, "- [%s](%s) — %s\n", path.Base(sub.Path)+"/", uriDePasta(sub.Path), contarNotas(sub.NotesRecursive))
		}
	}
	if len(arvore.Notes) > 0 {
		b.WriteString("\n## Notas\n\n")
		for _, n := range arvore.Notes {
			fmt.Fprintf(&b, "- [%s](%s)\n", folhaDaNota(n), resourceURI(vault.CanonicalPath(n.Path)))
		}
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: uri, MIMEType: mimeNota, Text: b.String()},
		},
	}, nil
}

func contarNotas(n int) string {
	if n == 1 {
		return "1 nota"
	}
	return fmt.Sprintf("%d notas", n)
}
