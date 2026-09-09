//go:build !windows

package instalar

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Fora do Windows nao ha registro: o PATH do usuario vem de um arquivo de
// perfil que a shell le na partida.
//
// O arquivo escolhido e ~/.profile, e nao ~/.bashrc ou ~/.zshrc, por uma razao
// concreta: escrever no arquivo de UMA shell obriga a adivinhar qual shell o
// usuario usa, e adivinhar errado deixa a instalacao silenciosamente sem PATH.
// ~/.profile e lido por sh, bash e (com --login) zsh. Quem usa fish ou nushell
// recebe a linha para colar, no aviso.
const marcaDoBloco = "# gobsidian (adicionado pelo instalador)"

func arquivoDePerfil() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolvendo o home do usuario: %w", err)
	}
	return filepath.Join(home, ".profile"), nil
}

// AdicionarAoPath acrescenta um bloco marcado ao ~/.profile.
//
// O bloco e MARCADO para que RemoverDoPath saiba exatamente o que tirar. Um
// instalador que acrescenta linha sem marca e um instalador que nunca consegue
// desfazer o que fez.
func AdicionarAoPath(dir string) (mudou bool, err error) {
	perfil, err := arquivoDePerfil()
	if err != nil {
		return false, err
	}

	conteudo, err := os.ReadFile(perfil)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("lendo %s: %w", perfil, err)
	}
	// Comparacao de DIRETORIO, e nao substring: "~/.local/bin" e
	// "~/.local/bin/" sao o mesmo lugar, e um substring casaria tambem com
	// "~/.local/bin-antigo". Sem isso o bloco e acrescentado a cada execucao.
	if jaTemOBloco(string(conteudo), dir) {
		return false, nil
	}

	bloco := fmt.Sprintf("\n%s\nexport PATH=\"%s:$PATH\"\n", marcaDoBloco, dir)
	f, err := os.OpenFile(perfil, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return false, fmt.Errorf("abrindo %s: %w", perfil, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(bloco); err != nil {
		return false, fmt.Errorf("gravando em %s: %w", perfil, err)
	}
	return true, nil
}

// RemoverDoPath tira o bloco marcado do ~/.profile, e SO ele: linhas que o
// usuario escreveu ficam.
func RemoverDoPath(dir string) (mudou bool, err error) {
	perfil, err := arquivoDePerfil()
	if err != nil {
		return false, err
	}
	f, err := os.Open(perfil)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("lendo %s: %w", perfil, err)
	}

	var mantidas []string
	pularProxima := false
	s := bufio.NewScanner(f)
	for s.Scan() {
		linha := s.Text()
		if pularProxima {
			pularProxima = false
			if strings.Contains(linha, dir) {
				mudou = true
				continue
			}
		}
		if strings.TrimSpace(linha) == marcaDoBloco {
			pularProxima = true
			mudou = true
			continue
		}
		mantidas = append(mantidas, linha)
	}
	_ = f.Close()
	if err := s.Err(); err != nil {
		return false, fmt.Errorf("lendo %s: %w", perfil, err)
	}
	if !mudou {
		return false, nil
	}

	saida := strings.Join(mantidas, "\n") + "\n"
	if err := os.WriteFile(perfil, []byte(saida), 0o600); err != nil {
		return false, fmt.Errorf("gravando %s: %w", perfil, err)
	}
	return true, nil
}

// AvisoDePath e o que o usuario precisa saber depois de o PATH mudar.
func AvisoDePath() string {
	return "abra um terminal NOVO, ou rode `source ~/.profile`; fish e nushell leem outro arquivo e precisam da linha a mao"
}

// DiretorioPadrao e onde o binario vai sem elevacao (decisao D-04).
//
// ~/.local/bin e o caminho por usuario da especificacao XDG, e as distribuicoes
// recentes ja o poem no PATH por padrao. /usr/local/bin exigiria sudo.
func DiretorioPadrao() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "gobsidian")
	}
	return filepath.Join(home, ".local", "bin")
}

// NomeDoExecutavel e o nome do arquivo instalado.
const NomeDoExecutavel = "gobsidian"

// jaTemOBloco diz se o perfil ja exporta dir num bloco nosso.
func jaTemOBloco(conteudo, dir string) bool {
	if !strings.Contains(conteudo, marcaDoBloco) {
		return false
	}
	for _, linha := range strings.Split(conteudo, "\n") {
		limpa := strings.TrimSpace(linha)
		if !strings.HasPrefix(limpa, "export PATH=") {
			continue
		}
		corpo := strings.Trim(strings.TrimPrefix(limpa, "export PATH="), `"`)
		for _, parte := range strings.Split(corpo, ":") {
			if mesmoDiretorio(parte, dir) {
				return true
			}
		}
	}
	return false
}
