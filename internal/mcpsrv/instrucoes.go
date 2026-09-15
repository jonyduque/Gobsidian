package mcpsrv

import (
	_ "embed"
	"path/filepath"
	"strings"

	"github.com/jonyduque/Gobsidian/internal/config"
)

// instrucoesBase e o texto que vai em `instructions` no initialize.
//
// Ate 2026-09-15 esse texto so existia em docs/PROMPT.md, e o usuario o colava
// a mao em cada host. Ele mora num arquivo, e nao numa constante, para ser a
// UNICA copia: docs/PROMPT.md aponta para ele, e scripts/check_prompt.ps1
// confere contra ele os conjuntos fechados que internal/service cobra.
//
//go:embed instrucoes.txt
var instrucoesBase string

// avisoSomenteLeitura substitui a frase fixa que o prompt manual trazia ("o
// cofre pode estar em modo somente-leitura"), que valia para todo cofre e nao
// dizia nada sobre este.
const avisoSomenteLeitura = `SOMENTE LEITURA
- Este cofre esta em modo somente-leitura nesta sessao: as tools de escrita
  nao existem. Nao prometa editar, criar, mover ou apagar notas.

`

// montarInstrucoes compoe o texto da sessao: o nome do cofre, o aviso de
// somente-leitura quando ele vale, e as regras de uso.
//
// O nome vem primeiro porque um usuario com varios cofres tem varios servidores
// gobsidian no mesmo host, e o modelo precisa saber qual e qual.
func montarInstrucoes(cfg config.Config) string {
	var b strings.Builder
	if cfg.VaultPath != "" {
		b.WriteString("Cofre: " + filepath.Base(cfg.VaultPath) + "\n\n")
	}
	if cfg.ReadOnly {
		b.WriteString(avisoSomenteLeitura)
	}
	b.WriteString(instrucoesBase)
	return b.String()
}
