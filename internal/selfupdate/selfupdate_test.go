package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// transporteFalso devolve bytes de memoria.
//
// Nenhum teste deste pacote importa net, e isso e requisito e nao estilo: a
// RNF-30 e sobre o que o produto abre, e um teste que sobe servidor para
// exercitar o cliente testaria o servidor junto -- alem de violar, no proprio
// pacote da excecao, a regra que ele existe para respeitar.
type transporteFalso struct {
	corpos   map[string]string
	buscadas []string
	erroFixo error
}

func (t *transporteFalso) Buscar(_ context.Context, url string) (io.ReadCloser, error) {
	t.buscadas = append(t.buscadas, url)
	if t.erroFixo != nil {
		return nil, t.erroFixo
	}
	corpo, ok := t.corpos[url]
	if !ok {
		return nil, fmt.Errorf("o teste nao preparou resposta para %s", url)
	}
	return io.NopCloser(strings.NewReader(corpo)), nil
}

// TestValidarHostRecusaOQueNaoEstaNaLista e a metade que a analise estatica
// NAO cobre.
//
// tools/netcheck prova que nenhum LITERAL de URL escrito neste pacote sai da
// lista. Ele nao prova nada sobre uma URL que chega em tempo de execucao -- e a
// URL de download vem do JSON da API, que e exatamente isso. Sem esta guarda,
// uma resposta adulterada mandaria o instalador buscar o binario onde quisesse.
func TestValidarHostRecusaOQueNaoEstaNaLista(t *testing.T) {
	casos := []struct {
		url     string
		aceitar bool
		porQue  string
	}{
		{HostDaAPI + "/repos/x/y/releases/latest", true, "a propria API"},
		{HostDeDownload + "/qualquer/caminho", true, "o host de download dos ativos"},
		{HostDoRepo + "/dono/projeto/releases", true, "o repositorio"},
		{"https://exemplo.invalid/payload", false, "host que nao esta na lista"},
		{"http://api.github.com/repos", false, "sem TLS"},
		{
			"https://api.github.com.exemplo.invalid/repos", false,
			"prefixo de TEXTO do host permitido, mas outro dominio -- e por isso que a comparacao exige o '/'",
		},
		{"", false, "vazio"},
	}

	for _, c := range casos {
		err := ValidarHost(c.url)
		if c.aceitar && err != nil {
			t.Errorf("ValidarHost(%q) recusou (%s): %v", c.url, c.porQue, err)
		}
		if !c.aceitar {
			if err == nil {
				t.Errorf("ValidarHost(%q) aceitou, e nao devia (%s)", c.url, c.porQue)
			} else if !errors.Is(err, ErrHostProibido) {
				t.Errorf("ValidarHost(%q) recusou com erro que nao e ErrHostProibido: %v", c.url, err)
			}
		}
	}
}

// TestUltimaVersaoDescartaAtivoComHostForaDaLista: um ativo cuja URL nao passa
// e descartado ao LER o release, e nao na hora de baixar -- assim ele nunca
// chega a aparecer como opcao.
func TestUltimaVersaoDescartaAtivoComHostForaDaLista(t *testing.T) {
	api := HostDaAPI + "/repos/dono/proj/releases/latest"
	tr := &transporteFalso{corpos: map[string]string{
		api: `{"tag_name":"v9.9.9","assets":[
			{"name":"bom.exe","browser_download_url":"` + HostDeDownload + `/bom"},
			{"name":"mau.exe","browser_download_url":"https://exemplo.invalid/mau"}
		]}`,
	}}

	r, err := UltimaVersao(context.Background(), tr, "dono/proj")
	if err != nil {
		t.Fatalf("UltimaVersao() error = %v", err)
	}
	if r.Tag != "v9.9.9" {
		t.Errorf("Tag = %q, esperado v9.9.9", r.Tag)
	}
	if _, ok := r.Ativos["bom.exe"]; !ok {
		t.Error("o ativo com host permitido foi descartado")
	}
	if _, ok := r.Ativos["mau.exe"]; ok {
		t.Error("o ativo com host FORA da lista sobreviveu; ele viraria uma opcao de download")
	}
}

// TestBaixarConfereOHashEAbortaNaDivergencia e a decisao D-03 do dono: quem
// confere e o binario VELHO, sobre o NOVO. Um binario que nao confere nao e
// instalado.
func TestBaixarConfereOHashEAbortaNaDivergencia(t *testing.T) {
	conteudo := "binario novo, de mentira"
	soma := sha256.Sum256([]byte(conteudo))
	somaCerta := hex.EncodeToString(soma[:])

	montar := func(somaPublicada string) (*transporteFalso, Release) {
		urlBin := HostDeDownload + "/gobsidian.exe"
		urlSomas := HostDeDownload + "/" + NomeDoArquivoDeSomas
		tr := &transporteFalso{corpos: map[string]string{
			urlBin:   conteudo,
			urlSomas: somaPublicada + "  gobsidian.exe\n",
		}}
		r := Release{Tag: "v9.9.9", Ativos: map[string]string{
			"gobsidian.exe":      urlBin,
			NomeDoArquivoDeSomas: urlSomas,
		}}
		return tr, r
	}

	t.Run("soma confere", func(t *testing.T) {
		tr, r := montar(somaCerta)
		destino := filepath.Join(t.TempDir(), "gobsidian.exe")
		if err := Baixar(context.Background(), tr, r, "gobsidian.exe", destino); err != nil {
			t.Fatalf("Baixar() error = %v", err)
		}
		lido, err := os.ReadFile(destino)
		if err != nil || string(lido) != conteudo {
			t.Fatalf("o arquivo gravado nao confere: %v / %q", err, lido)
		}
	})

	t.Run("soma diverge", func(t *testing.T) {
		tr, r := montar(strings.Repeat("0", 64))
		destino := filepath.Join(t.TempDir(), "gobsidian.exe")
		err := Baixar(context.Background(), tr, r, "gobsidian.exe", destino)
		if !errors.Is(err, ErrHashDivergente) {
			t.Fatalf("Baixar() com soma errada devolveu %v, esperado ErrHashDivergente", err)
		}
		// E, sobretudo, NAO deixou o arquivo no destino.
		if _, err := os.Stat(destino); !os.IsNotExist(err) {
			t.Fatalf("um binario que nao confere ficou no destino: %v", err)
		}
	})

	t.Run("release sem SHA256SUMS aborta", func(t *testing.T) {
		urlBin := HostDeDownload + "/gobsidian.exe"
		tr := &transporteFalso{corpos: map[string]string{urlBin: conteudo}}
		r := Release{Tag: "v0.1.0", Ativos: map[string]string{"gobsidian.exe": urlBin}}
		destino := filepath.Join(t.TempDir(), "gobsidian.exe")

		err := Baixar(context.Background(), tr, r, "gobsidian.exe", destino)
		if err == nil {
			t.Fatal("Baixar() aceitou um release sem somas publicadas")
		}
		if !strings.Contains(err.Error(), NomeDoArquivoDeSomas) {
			t.Errorf("o erro nao nomeia o arquivo que falta: %v", err)
		}
		if _, err := os.Stat(destino); !os.IsNotExist(err) {
			t.Error("gravou o binario mesmo sem ter o que conferir")
		}
	})
}

// TestNadaSaiSemPassarPorValidarHost prova que a guarda esta ANTES do
// transporte, e nao depois: com uma URL proibida, o transporte nao chega a ser
// chamado. Um transporte que ja discou nao pode desfazer a conexao.
func TestNadaSaiSemPassarPorValidarHost(t *testing.T) {
	tr := &transporteFalso{corpos: map[string]string{}}
	r := Release{Tag: "v1", Ativos: map[string]string{
		"x.exe":              "https://exemplo.invalid/x",
		NomeDoArquivoDeSomas: "https://exemplo.invalid/sums",
	}}

	_ = Baixar(context.Background(), tr, r, "x.exe", filepath.Join(t.TempDir(), "x.exe"))

	if len(tr.buscadas) != 0 {
		t.Fatalf("o transporte foi chamado com host proibido: %v", tr.buscadas)
	}
}

// TestListaDeHostsNaoDivergiuDoAnalisador: ha DUAS listas dos mesmos hosts --
// uma em tools/netcheck, verificada em tempo de compilacao, e outra aqui,
// verificada em tempo de execucao. Elas cobrem metades diferentes e por isso
// as duas existem; este teste e o que impede que uma mude sozinha.
//
// A conta e feita sobre o arquivo do analisador, e nao sobre uma terceira
// copia da lista escrita neste teste -- uma terceira copia so mudaria de dois
// para tres o numero de coisas que podem divergir.
func TestListaDeHostsNaoDivergiuDoAnalisador(t *testing.T) {
	fonte, err := os.ReadFile(filepath.Join("..", "..", "tools", "netcheck", "netcheck.go"))
	if err != nil {
		t.Fatalf("lendo o analisador: %v", err)
	}
	for _, h := range HostsPermitidos() {
		if !strings.Contains(string(fonte), `"`+h+`"`) {
			t.Errorf("o host %q e aceito em tempo de execucao mas o analisador nao o conhece", h)
		}
	}
}
