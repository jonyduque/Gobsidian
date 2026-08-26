//go:build windows

package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// travaLeitura impede a leitura do arquivo, como um aplicativo que o mantém
// aberto com acesso exclusivo.
//
// Medido em 2026-08-26: `GENERIC_READ` com `share=0` bloqueia só a remoção;
// é preciso `GENERIC_READ|GENERIC_WRITE` para bloquear a leitura.
func travaLeitura(t *testing.T, abs string) {
	t.Helper()
	p, err := windows.UTF16PtrFromString(abs)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Skipf("nao foi possivel travar %q para leitura: %v", abs, err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
}

// TestReplaceComErroDeLeituraNaoDeixaANotaForaDoIndice cobre o A3.
//
// `Replace` removia TODAS as contribuições antigas ANTES do I/O. Se a leitura
// falhasse, a nota ficava fora dos metadados sem republish — e sem
// reprocessamento dos citantes, então os links para ela continuavam com
// `state=ok` apontando para um caminho que `Get` não resolve mais.
//
// Pior no conjunto: `apply.go`, vendo o erro, pula `searchInv.Update`. A busca
// mantém o documento velho e os metadados não têm mais a nota; `service.Search`
// descarta posting sem metadado, e **a nota some das respostas** até o próximo
// evento, reconciliação ou boot.
//
// O contrato que este teste fixa: um erro TRANSITÓRIO de leitura não pode mudar
// o estado do índice. Ou a nota continua como estava, ou sai — nunca um meio
// termo em que os metadados e a busca discordam.
func TestReplaceComErroDeLeituraNaoDeixaANotaForaDoIndice(t *testing.T) {
	dir := t.TempDir()
	nota := filepath.Join(dir, "nota.md")
	if err := os.WriteFile(nota, []byte("# Titulo\n\ncorpo.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "citante.md"),
		[]byte("# Citante\n\nver [[nota]].\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	canon := vault.CanonicalPath("nota.md")
	if _, ok := idx.Get(canon); !ok {
		t.Fatal("cenario invalido: a nota nem entrou no indice")
	}

	// Erro TRANSITORIO: o arquivo continua existindo, só não pode ser lido.
	travaLeitura(t, nota)

	errReplace := idx.Replace(context.Background(), v, canon)
	if errReplace == nil {
		t.Fatal("Replace devolveu nil com a leitura travada; o cenario nao exercitou nada")
	}

	// O estado tem de ser CONSISTENTE. O defeito era a nota sair dos metadados
	// enquanto tudo o mais continuava achando que ela estava lá.
	_, aindaNoIndice := idx.Get(canon)
	if !aindaNoIndice {
		t.Errorf("a nota saiu do indice por um erro TRANSITORIO de leitura: " +
			"o arquivo continua no disco, e o indice agora discorda dele")
	}
}

// TestReplaceComNotaRemovidaEntreOEventoEALeitura fixa a política da janela que
// o redesenho em duas fases abre.
//
// Com o I/O fora do lock, o arquivo pode sumir entre a leitura e a publicação.
// A política é: confirmar sob o lock; se sumiu, tratar como remoção. Sem isso,
// uma nota apagada voltaria ao índice e ficaria lá até o próximo evento.
func TestReplaceComNotaRemovidaEntreOEventoEALeitura(t *testing.T) {
	dir := t.TempDir()
	nota := filepath.Join(dir, "some.md")
	if err := os.WriteFile(nota, []byte("# Some\n\ncorpo.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	canon := vault.CanonicalPath("some.md")
	if _, ok := idx.Get(canon); !ok {
		t.Fatal("cenario invalido: a nota nem entrou no indice")
	}

	if err := os.Remove(nota); err != nil {
		t.Fatalf("removendo a nota: %v", err)
	}

	if err := idx.Replace(context.Background(), v, canon); err != nil {
		t.Fatalf("Replace de nota ausente devia ser nil, nao erro: %v", err)
	}
	if _, ok := idx.Get(canon); ok {
		t.Error("a nota apagada continua no indice depois do Replace")
	}
}

// TestReplacePublicaAliasComoOBuild cobre o M7.
//
// `Replace` reimplementava a publicação: seis derivações refeitas à mão —
// notes, lowerPath, byName, byAlias, tags e citantes — enquanto
// `publishNoteLocked` já fazia a mesma coisa para o `Build`. Duas contas do
// mesmo fato concordavam por coincidência, e é o padrão exato que produziu o
// bug `[[STJ]]`.
//
// O teste compara as DUAS construções campo a campo pelo efeito observável:
// um alias criado por Replace tem de resolver igual a um criado pelo Build.
func TestReplacePublicaAliasComoOBuild(t *testing.T) {
	const conteudo = "---\naliases: [Apelido]\n---\n\n# Titulo\n\ncorpo.\n"

	const citante = "# Citante\n\nver [[Apelido]] aqui.\n"

	// Construção A: as duas notas presentes no Build.
	dirA := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirA, "n.md"), []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirA, "citante.md"), []byte(citante), 0o644); err != nil {
		t.Fatal(err)
	}
	vA, err := vault.New(dirA)
	if err != nil {
		t.Fatal(err)
	}
	idxA := index.New()
	if err := idxA.Build(context.Background(), vA); err != nil {
		t.Fatal(err)
	}

	// Construção B: só o citante no Build; o alvo com alias chega DEPOIS, por
	// Replace — que é o caminho do watcher.
	dirB := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirB, "citante.md"), []byte(citante), 0o644); err != nil {
		t.Fatal(err)
	}
	vB, err := vault.New(dirB)
	if err != nil {
		t.Fatal(err)
	}
	idxB := index.New()
	if err := idxB.Build(context.Background(), vB); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "n.md"), []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := idxB.Replace(context.Background(), vB, vault.CanonicalPath("n.md")); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	// Efeito observável: o link por alias resolve, e portanto o alvo tem
	// backlink. Comparar as duas construções é o que prova a paridade — não
	// conferir um valor escrito à mão.
	alvo := vault.CanonicalPath("n.md")
	blA := idxA.Backlinks(alvo)
	blB := idxB.Backlinks(alvo)

	if len(blA) == 0 {
		t.Fatalf("cenario invalido: nem o Build resolveu o alias (backlinks=%d)", len(blA))
	}
	if len(blB) != len(blA) {
		t.Errorf("as duas construcoes discordam: Build=%d backlink(s), Replace=%d; "+
			"o alias publicado por Replace nao resolve como o do Build",
			len(blA), len(blB))
	}
}
