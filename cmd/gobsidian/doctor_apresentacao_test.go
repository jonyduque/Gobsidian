package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/doctor"
	"github.com/jonyduque/Gobsidian/internal/instalar"
)

// TestRelatarVerificacoesAgrupaEResume: ate 2026-09-16 as dezesseis linhas do
// relatorio saiam numa lista unica, cada uma com o detalhe numa segunda linha.
// Quem abre o `doctor` esta procurando UMA coisa, e achar exigia ler tudo.
func TestRelatarVerificacoesAgrupaEResume(t *testing.T) {
	t.Setenv(console.VarDeEstilo, "1")

	var buf bytes.Buffer
	con := console.NewPlain(&buf)
	relatarVerificacoes(con, []doctor.Result{
		{Name: "raiz do cofre existe", Status: doctor.StatusOK, Grupo: doctor.GrupoCofre},
		{Name: "contagem de notas", Status: doctor.StatusOK, Detail: "3257 notas", Grupo: doctor.GrupoCofre},
		{Name: "espaço em disco", Status: doctor.StatusFail, Detail: "8 MB livres", Grupo: doctor.GrupoCache},
		{Name: "daemon respondendo", Status: doctor.StatusWarn, Detail: "handshake falhou", Grupo: doctor.GrupoDaemon},
	})
	saida := buf.String()

	// As secoes saem na ordem, e cada uma aparece uma vez.
	iCofre, iCache, iDaemon := strings.Index(saida, doctor.GrupoCofre), strings.Index(saida, doctor.GrupoCache), strings.Index(saida, doctor.GrupoDaemon)
	if iCofre < 0 || iCache < 0 || iDaemon < 0 {
		t.Fatalf("faltam secoes na saida:\n%s", saida)
	}
	if iCofre >= iCache || iCache >= iDaemon {
		t.Errorf("secoes fora de ordem (cofre=%d, cache=%d, daemon=%d):\n%s", iCofre, iCache, iDaemon, saida)
	}
	// Nenhum grupo vazio: "Windows" nao foi passado e nao pode aparecer.
	if strings.Contains(saida, doctor.GrupoSO) {
		t.Errorf("secao sem nenhuma verificacao apareceu:\n%s", saida)
	}

	// Detalhe curto de um OK vai na PROPRIA linha; o de um aviso, embaixo.
	linhaDaContagem := linhaCom(t, saida, "contagem de notas")
	if !strings.Contains(linhaDaContagem, "3257 notas") {
		t.Errorf("o detalhe de uma linha OK nao foi para a propria linha: %q", linhaDaContagem)
	}
	linhaDoDaemon := linhaCom(t, saida, "daemon respondendo")
	if strings.Contains(linhaDoDaemon, "handshake falhou") {
		t.Errorf("o detalhe de um aviso subiu para a linha do nome: %q", linhaDoDaemon)
	}
	if !strings.Contains(saida, "     handshake falhou") {
		t.Errorf("o detalhe do aviso sumiu:\n%s", saida)
	}

	// O resumo e o que responde "e isso tudo esta bem?" sem recontar as linhas.
	if !strings.Contains(saida, "4 verificações: 2 ok, 1 aviso(s), 1 falha(s)") {
		t.Errorf("resumo ausente ou errado:\n%s", saida)
	}
}

// TestLinhasPorCofreResumeOsProcessos: 28 processos viravam 28 linhas de pid,
// e a pergunta que alguem faz olhando aquilo -- quantos servem este cofre, e
// quantos gravam o cache -- exigia contar a mao. Medido em 2026-09-15.
func TestLinhasPorCofreResumeOsProcessos(t *testing.T) {
	t.Setenv(console.VarDeEstilo, "1")

	var buf bytes.Buffer
	con := console.NewPlain(&buf)
	p := func(pid int, cofre, papel, modo, versao string) instalar.Presenca {
		return instalar.Presenca{PID: pid, Cofre: cofre, Papel: papel, Modo: modo, Versao: versao}
	}
	linhas := linhasPorCofre(con, []instalar.Presenca{
		p(1, `C:\Obsidian\Estudo`, "daemon", instalar.ModoDaemon, "v1.9.0"),
		p(2, `C:\Obsidian\Estudo`, "serve", instalar.ModoPonte, "v1.9.0"),
		p(3, `C:\Obsidian\Estudo`, "serve", instalar.ModoPonte, "v1.9.0"),
		p(4, `C:\Obsidian\Oral`, "serve", instalar.ModoEmProcesso, "v1.8.1"),
		p(5, `C:\Obsidian\Oral`, "serve", "", ""),
	})

	if len(linhas) != 2 {
		t.Fatalf("esperava uma linha por cofre, recebi %d:\n%s", len(linhas), strings.Join(linhas, "\n"))
	}
	estudo, oral := linhas[0], linhas[1]
	if !strings.Contains(estudo, "Estudo") || !strings.Contains(estudo, "1 daemon") || !strings.Contains(estudo, "2 pontes") {
		t.Errorf("a linha do Estudo nao conta os modos, ou nao poe o plural: %q", estudo)
	}
	if !strings.Contains(estudo, "v1.9.0") {
		t.Errorf("a linha do Estudo nao diz a versao: %q", estudo)
	}
	if !strings.Contains(oral, "1 servidor em processo") || !strings.Contains(oral, "1 modo não registrado") {
		t.Errorf("a linha do Oral nao separa o processo de versao anterior: %q", oral)
	}
}

func linhaCom(t *testing.T, saida, trecho string) string {
	t.Helper()
	for _, l := range strings.Split(saida, "\n") {
		if strings.Contains(l, trecho) {
			return l
		}
	}
	t.Fatalf("nenhuma linha com %q em:\n%s", trecho, saida)
	return ""
}
