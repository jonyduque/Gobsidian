package instalar

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jonyduque/Gobsidian/internal/hosts"
	"github.com/jonyduque/Gobsidian/internal/text"
)

// EntradaDeOutroBinario e uma entrada do gobsidian num config de host cujo
// comando nao e o binario instalado.
type EntradaDeOutroBinario struct {
	Host    string
	Chave   string
	Comando string
	// Executavel e o arquivo que o comando resolve, e vazio quando ele nao
	// resolve para arquivo nenhum -- o host falharia ao iniciar.
	Executavel string
}

// EntradasDeOutroBinario lista as entradas nossas, nos configs de host de
// ARQUIVO, que nao rodam o binario instalado.
//
// Medido em 2026-09-14: o Claude Desktop rodava v1.8.1 do caminho instalado, e
// o Claude Code rodava v1.5.1 de C:\Program Files\gobsidian. Nada dizia isso --
// o `doctor` so foi achar os processos antigos pela lista do sistema (G8), e
// sem dizer de onde vinham.
//
// So enxerga host de arquivo, pelo mesmo motivo de ConfiguracaoAtual: a
// configuracao do Claude Code, do Gemini CLI, do Codex e do VS Code e do
// proprio CLI. Para esses, a lista de processos sem presenca e o que sobra.
//
// Nao reescreve nada. Quem reconfigura e `install`.
func EntradasDeOutroBinario(amb hosts.Ambiente, instalado string) []EntradaDeOutroBinario {
	var saida []EntradaDeOutroBinario
	for _, h := range hosts.Detectar(amb) {
		for _, en := range h.EntradasAtuais(amb) {
			executavel := resolverComando(en.Command)
			if executavel != "" && mesmoExecutavel(executavel, instalado) {
				continue
			}
			saida = append(saida, EntradaDeOutroBinario{
				Host:       h.Nome,
				Chave:      en.Chave,
				Comando:    en.Command,
				Executavel: executavel,
			})
		}
	}
	return saida
}

// VersaoDoExecutavel devolve a versao de um executavel pela presenca de um
// processo vivo dele, ou "" quando nenhum processo dele registrou presenca.
//
// Nao roda o executavel para perguntar. `doctor` e diagnostico, e um binario
// de outra instalacao, rodado por ele, escreve log e runtime do jeito da
// versao dele -- um diagnostico que muda o sistema so por ter sido consultado
// nao e diagnostico. O custo e que binario anterior a presenca, o caso medido
// do v1.5.1, sai sem versao.
func VersaoDoExecutavel(executavel string, vivos []Presenca, processos []ProcessoDoSistema) string {
	versaoDoPID := make(map[int]string, len(vivos))
	for _, p := range vivos {
		versaoDoPID[p.PID] = p.Versao
	}
	for _, p := range processos {
		v := versaoDoPID[p.PID]
		if v != "" && mesmoExecutavel(p.Executavel, executavel) {
			return v
		}
	}
	return ""
}

// resolverComando e o arquivo que um host acharia para o comando, ou "".
func resolverComando(comando string) string {
	if comando == "" {
		return ""
	}
	caminho, err := exec.LookPath(comando)
	if err != nil {
		return ""
	}
	abs, err := filepath.Abs(caminho)
	if err != nil {
		return caminho
	}
	return abs
}

// mesmoExecutavel compara dois caminhos de executavel. Por os.SameFile quando
// os dois existem, porque ela atravessa link e grafia (a mesma escolha de
// EstaInstalado); pela chave de caminho quando um deles nao existe mais.
func mesmoExecutavel(a, b string) bool {
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	if errA == nil && errB == nil {
		return os.SameFile(ia, ib)
	}
	return text.ChaveDeCaminho(a) == text.ChaveDeCaminho(b)
}
