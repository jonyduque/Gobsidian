package vault_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// criaSymlink cria um link simbolico e devolve false quando o sistema recusa
// por PERMISSAO.
//
// A distincao importa. No Windows, criar link simbolico exige privilegio
// elevado ou o Modo de Desenvolvedor ligado, e numa maquina comum a chamada
// falha. Pular nesse caso e legitimo; pular em silencio, nao — um t.Skip
// permanente vira cobertura fantasma, e este projeto ja teve tres testes que
// reportavam cobertura inexistente. Por isso a mensagem do skip diz que foi
// permissao, e QUALQUER outro erro e falha.
func criaSymlink(t *testing.T, alvo, link string) bool {
	t.Helper()
	err := os.Symlink(alvo, link)
	if err == nil {
		return true
	}
	if os.IsPermission(err) || errosDePrivilegioDeSymlink(err) {
		t.Skipf("RNF-32 NAO VERIFICADO nesta maquina: criacao de link simbolico "+
			"recusada por PERMISSAO (%v). No Windows exige privilegio elevado ou "+
			"Modo de Desenvolvedor. Rode elevado para exercitar este teste.", err)
		return false
	}
	t.Fatalf("os.Symlink falhou por motivo que NAO e permissao: %v", err)
	return false
}

// TestWalkNaoSegueSymlink fixa o RNF-32: vault.Walk não atravessa link
// simbólico.
//
// Até 2026-08-02 a propriedade valia "por construção" — `filepath.WalkDir` não
// segue links — e tinha apenas teste indireto na Task 7. Propriedade que vale
// por construção continua valendo até alguém trocar a construção; o que este
// teste fixa é que trocar `WalkDir` por algo que siga links passa a quebrar
// aqui, em vez de abrir uma varredura para fora do cofre em silêncio.
//
// O cenário é o que importa: o alvo do link fica FORA da raiz e contém uma
// nota. Se a varredura seguisse o link, essa nota apareceria — e o
// confinamento de caminho, que é a garantia inteira, teria sido furado por uma
// camada que ninguém suspeitava.
func TestWalkNaoSegueSymlink(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "cofre")
	fora := filepath.Join(base, "fora")

	for _, d := range []string{root, fora} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "dentro.md"), []byte("# Dentro"), 0o644); err != nil {
		t.Fatalf("WriteFile dentro.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fora, "vazada.md"), []byte("# Vazada"), 0o644); err != nil {
		t.Fatalf("WriteFile vazada.md: %v", err)
	}

	if !criaSymlink(t, fora, filepath.Join(root, "atalho")) {
		return
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	var vistos []string
	if err := v.Walk(context.Background(), func(e vault.Entry) error {
		vistos = append(vistos, string(e.Path))
		return nil
	}); err != nil {
		t.Fatalf("Walk: %v", err)
	}

	// Guarda de cenario: se dentro.md nao aparecer, a varredura nao andou e a
	// ausencia de vazada.md nao prova nada.
	var achouDentro bool
	for _, p := range vistos {
		if p == "dentro.md" {
			achouDentro = true
		}
		if filepath.Base(p) == "vazada.md" {
			t.Errorf("Walk atravessou o link simbolico e entregou %q — RNF-32 violado: "+
				"a varredura saiu da raiz do cofre", p)
		}
	}
	if !achouDentro {
		t.Fatalf("Walk nao entregou dentro.md; entradas vistas: %v — o cenario nao "+
			"exercitou nada e a ausencia de vazada.md nao significa nada", vistos)
	}
}

// segredoFora e o conteudo do alvo de fora do cofre. Se ele aparecer numa
// leitura, o confinamento vazou.
const segredoFora = "conteudo-fora-do-cofre-que-nao-deve-vazar"

// montaSymlinkDeArquivo cria, dentro do cofre, um `.md` que aponta para um
// arquivo FORA dele.
//
// TestWalkNaoSegueSymlink, acima, cobre o symlink de DIRETORIO — que WalkDir
// nunca atravessou. Este cobre o de ARQUIVO, que era o buraco: um symlink
// chamado nota.md passava nas duas camadas lexicas de confinamento, entrava no
// indice, e note_read devolvia conteudo arbitrario do disco pelo canal MCP.
func montaSymlinkDeArquivo(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	cofre := filepath.Join(base, "cofre")
	if err := os.MkdirAll(cofre, 0o755); err != nil {
		t.Fatal(err)
	}
	alvo := filepath.Join(base, "fora.txt")
	if err := os.WriteFile(alvo, []byte(segredoFora), 0o644); err != nil {
		t.Fatal(err)
	}
	if !criaSymlink(t, alvo, filepath.Join(cofre, "nota.md")) {
		return ""
	}
	return cofre
}

// TestLeituraRecusaSymlinkPorPadrao cobre o confinamento na camada que ABRE o
// arquivo.
//
// As duas camadas existentes sao lexicas: validateLocal e Canonicalize, e
// nenhuma consulta o disco. path.go documenta o limite e delega a "camada que
// abre o arquivo" — que, ate 2026-08-26, era um os.Open puro e nao exercia
// checagem nenhuma.
func TestLeituraRecusaSymlinkPorPadrao(t *testing.T) {
	cofre := montaSymlinkDeArquivo(t)
	if cofre == "" {
		return
	}

	v, err := vault.New(cofre)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	p := vault.CanonicalPath("nota.md")

	dados, err := v.ReadAll(context.Background(), p)
	if err == nil {
		t.Errorf("ReadAll seguiu symlink por padrao e devolveu %q", string(dados))
	} else if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("o erro nao nomeia a causa: %v", err)
	}

	if f, errOpen := v.Open(p); errOpen == nil {
		_ = f.Close()
		t.Error("Open seguiu symlink por padrao")
	}

	if _, errRange := v.ReadRange(context.Background(), p, 0, 10); errRange == nil {
		t.Error("ReadRange seguiu symlink por padrao")
	}
}

// TestWalkPulaSymlinkDeArquivo garante que o symlink nem chega ao indice, e que
// o descarte e VISIVEL.
//
// Recusar na leitura e indexar mesmo assim produziria uma nota que aparece em
// note_list e falha em note_read — pior que qualquer um dos dois isolado. E
// descarte silencioso repetiria a licao ja paga: cofre inacessivel e cofre
// vazio nao podem produzir a mesma resposta.
func TestWalkPulaSymlinkDeArquivo(t *testing.T) {
	cofre := montaSymlinkDeArquivo(t)
	if cofre == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(cofre, "real.md"), []byte("# Real\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(cofre)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	var vistos []string
	if err := v.Walk(context.Background(), func(e vault.Entry) error {
		vistos = append(vistos, string(e.Path))
		return nil
	}); err != nil {
		t.Fatalf("Walk: %v", err)
	}

	for _, p := range vistos {
		if p == "nota.md" {
			t.Error("Walk entregou o symlink de arquivo a indexacao")
		}
	}
	if len(vistos) != 1 || vistos[0] != "real.md" {
		t.Errorf("Walk viu %v, queria apenas [real.md]", vistos)
	}

	n, amostras := v.SkippedEntries()
	if n == 0 {
		t.Error("o symlink foi descartado em SILENCIO: SkippedEntries = 0")
	}
	if !strings.Contains(strings.Join(amostras, " "), "symlink") {
		t.Errorf("a amostra de descarte nao explica o motivo: %v", amostras)
	}
}

// TestSeguirSymlinksPreservaOComportamentoAntigo e o contrapeso da decisao do
// dono: recusar por padrao, mas NAO tirar a possibilidade.
//
// Symlinkar uma pasta externa para dentro do cofre e workflow legitimo e
// suportado pelo Obsidian. Sem esta saida, a correcao trocaria um risco
// hipotetico por uma regressao certa.
func TestSeguirSymlinksPreservaOComportamentoAntigo(t *testing.T) {
	cofre := montaSymlinkDeArquivo(t)
	if cofre == "" {
		return
	}

	v, err := vault.New(cofre, vault.SeguirSymlinks(true))
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	dados, err := v.ReadAll(context.Background(), vault.CanonicalPath("nota.md"))
	if err != nil {
		t.Fatalf("com SeguirSymlinks(true), ReadAll devia funcionar: %v", err)
	}
	if !strings.Contains(string(dados), segredoFora) {
		t.Errorf("o conteudo lido nao e o do alvo: %q", string(dados))
	}
}
