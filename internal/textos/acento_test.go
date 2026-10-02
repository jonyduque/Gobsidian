package textos_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// semAcento sao palavras que, em portugues, so existem com acento. Escritas
// sem ele num texto que o produto mostra, sao erro de redacao -- e a regra do
// projeto e que o texto sai com acento, e quem decide tira-lo e o console
// (console.adaptarTexto), so onde a code page nao aguenta.
//
// A lista evita as ambiguas: "esta" (esta/está), "e" (e/é), "pais"
// (pais/país) e afins ficam de fora, porque reprovariam texto certo.
var semAcento = regexp.MustCompile(`(?i)\b(` + strings.Join([]string{
	"nao", "indice", "indices", "ja", "ate", "sao", "apos", "alem", "tambem", "entao", "voce",
	"invalido", "invalida", "invalidos", "invalidas", "diretorio", "diretorios",
	"conteudo", "titulo", "titulos", "codigo", "pagina", "versao", "numero",
	"unico", "unica", "proprio", "propria", "possivel", "disponivel", "disponiveis",
	"ancora", "ancoras", "valido", "valida", "obrigatorio", "obrigatoria",
	"[a-z]+cao", "[a-z]+coes",
}, "|") + `)\b`)

// excecoes sao literais que casam a lista e NAO sao texto: identificador,
// nome de arquivo, valor de entrada, dado. A chave e "arquivo|literal"; o
// valor diz por que fica. Excecao nova entra com o motivo, como estas.
var excecoes = map[string]string{
	"cmd/gobsidian/ponte.go|versao-divergente":          "motivo de queda da ponte: identificador que o log e o OPERACAO.md citam por texto",
	"internal/console/cobra.go|secao":                   "nome de funcao do template de ajuda do cobra",
	"internal/console/estilo.go|nao":                    "valor aceito em GOBSIDIAN_ESTILO: entrada do usuario, sem acento de proposito",
	"internal/instalar/manifesto.go|instalacao.json":    "nome de arquivo no disco do usuario",
	"internal/instalar/trava_global.go|instalacao.lock": "nome de arquivo no disco do usuario",
	"internal/search/analyzer.go|sao":                   "sufixo do stemmer, aplicado a texto ja sem acento",
	"internal/search/persist_codec.go|totalPosicoes":    "rotulo de campo do codec, identificador",
	"internal/search/persist_codec.go|nPosicoes":        "rotulo de campo do codec, identificador",
}

// chamadasDeLog sao os metodos do slog. A mensagem e as chaves deles ficam
// sem acento de proposito: scripts/measure.ps1 casa "servidor pronto",
// scripts/test_orphans.ps1 casa "reason=", e o log e lido por grep, nao por
// gente.
var chamadasDeLog = map[string]bool{"Debug": true, "Info": true, "Warn": true, "Error": true, "Log": true, "With": true}

// TestTextoDoProdutoTemAcento le os literais de string do codigo de PRODUCAO
// em internal/ e cmd/ e reprova palavra que exige acento escrita sem ele.
//
// Existe porque o levantamento de 2026-09-27 (docs/TEXTOS.md, "Textos fora de
// internal/textos") cobriu instalar, hosts, selfupdate, config, doctor e
// console e deixou internal/service e internal/mcpsrv de fora -- e e de la que
// vem toda mensagem de erro das tools, a que o dono viu na CLI em 2026-10-01:
// "Nota ... nao encontrada no indice". Nenhum gate olhava; agora este olha.
//
// Comentario fica de fora (o codigo comenta sem acento por estilo), e tambem
// o literal que e argumento de slog (ver chamadasDeLog) e o de tag de struct,
// que nao e texto.
func TestTextoDoProdutoTemAcento(t *testing.T) {
	raiz, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var achados []string
	for _, base := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(raiz, base), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" || d.Name() == "vaulttest" || d.Name() == "vazamentotest" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			achados = append(achados, literaisSemAcento(t, raiz, p)...)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(achados)
	if len(achados) > 0 {
		t.Errorf("%d texto(s) do produto sem acento:\n%s", len(achados), strings.Join(achados, "\n"))
	}
}

func literaisSemAcento(t *testing.T, raiz, caminho string) []string {
	t.Helper()
	src, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, caminho, src, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Literais que sao argumento de slog, coletados antes para serem pulados.
	deLog := map[*ast.BasicLit]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !chamadasDeLog[sel.Sel.Name] {
			return true
		}
		// Todo literal DENTRO dos argumentos, e nao so o argumento direto: uma
		// mensagem longa e montada com + (index/update.go) e continua sendo de log.
		for _, a := range call.Args {
			ast.Inspect(a, func(m ast.Node) bool {
				if lit, ok := m.(*ast.BasicLit); ok {
					deLog[lit] = true
				}
				return true
			})
		}
		return true
	})

	// O literal da tag inteira e pulado: o que importa nele e so o valor de
	// jsonschema, conferido no proprio campo abaixo.
	tags := map[*ast.BasicLit]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if campo, ok := n.(*ast.Field); ok && campo.Tag != nil {
			tags[campo.Tag] = true
		}
		return true
	})

	rel, _ := filepath.Rel(raiz, caminho)
	var achados []string
	ast.Inspect(f, func(n ast.Node) bool {
		if campo, ok := n.(*ast.Field); ok && campo.Tag != nil {
			// Tag de struct: `json:"..."` nao e texto. A descricao que o
			// modelo le mora em `jsonschema:"..."`, e essa entra.
			if tag, err := strconv.Unquote(campo.Tag.Value); err == nil {
				if d := valorDaTag(tag, "jsonschema"); d != "" && semAcento.MatchString(d) {
					achados = append(achados, formatar(fset, rel, campo.Tag, d))
				}
			}
			return true
		}
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING || deLog[lit] || tags[lit] {
			return true
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil || !semAcento.MatchString(s) {
			return true
		}
		if _, ok := excecoes[filepath.ToSlash(rel)+"|"+s]; ok {
			return true
		}
		achados = append(achados, formatar(fset, rel, lit, s))
		return true
	})
	return achados
}

// valorDaTag le uma chave de tag de struct sem reflect.StructTag, que exige
// um valor ja tipado.
func valorDaTag(tag, chave string) string {
	i := strings.Index(tag, chave+`:"`)
	if i < 0 {
		return ""
	}
	resto := tag[i+len(chave)+2:]
	j := strings.Index(resto, `"`)
	if j < 0 {
		return ""
	}
	return resto[:j]
}

func formatar(fset *token.FileSet, rel string, n ast.Node, s string) string {
	palavras := strings.Join(semAcento.FindAllString(s, -1), ", ")
	return filepath.ToSlash(rel) + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) + ": [" + palavras + "] " + s
}
