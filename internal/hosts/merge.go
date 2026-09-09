// Package hosts sabe onde cada host de IA guarda a configuracao de servidores
// MCP, e como acrescentar o gobsidian a ela sem destruir o que ja estava la.
//
// Ate 2026-09-08 isso vivia em `install.ps1` (729 linhas) e em
// `installer/install.js` (1 009) -- a mesma tabela de nove hosts escrita duas
// vezes, em duas linguagens, sem um unico teste. E o arquivo que ele edita e
// um arquivo que o usuario editou a mao e nao tem copia.
//
// O pacote e FOLHA: recebe a raiz do sistema de arquivos e o comando a
// registrar por parametro, e nao conhece cofre, indice nem MCP. Ele escreve
// JSON generico; nao sabe o que e uma tool.
package hosts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ChaveDoServidor e o nome sob o qual o gobsidian aparece na configuracao de
// cada host. Uma conta so: a deteccao de "ja instalado", a substituicao e a
// remocao usam esta mesma constante.
const ChaveDoServidor = "gobsidian"

// SufixoDeBackup e o que Fundir deixa ao lado do arquivo antes de toca-lo.
//
// Config de host e coisa que o usuario editou a mao e nao tem copia. O backup
// e o unico caminho de volta se a fusao sair errada, e ele custa o tamanho de
// um arquivo de texto.
const SufixoDeBackup = ".gobsidian-backup"

// Entrada e o que se escreve sob mcpServers.<ChaveDoServidor>.
type Entrada struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// Fundir acrescenta (ou substitui) a entrada do gobsidian em caminho,
// PRESERVANDO todo o resto do arquivo.
//
// Sobrescrever apagaria as preferencias do host e os outros servidores MCP que
// o usuario ja tem -- o claude_desktop_config.json da maquina do dono carrega
// `preferences` e outro servidor ao lado. Esse comportamento ja existia no
// instalador em PowerShell; o que muda aqui e que agora ha teste.
//
// A decodificacao passa por map[string]json.RawMessage de proposito: campos
// que este pacote nao conhece atravessam a fusao BYTE A BYTE, sem passar por
// uma struct que os descartaria. Um "preferences" com forma inesperada tem de
// sair igual ao que entrou.
func Fundir(caminho string, e Entrada) error {
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		return fmt.Errorf("criando diretorio de %s: %w", caminho, err)
	}

	bruto, err := os.ReadFile(caminho)
	switch {
	case err == nil:
		if err := fazerBackup(caminho, bruto); err != nil {
			return err
		}
	case os.IsNotExist(err):
		bruto = nil
	default:
		return fmt.Errorf("lendo %s: %w", caminho, err)
	}

	doc := map[string]json.RawMessage{}
	if len(bruto) > 0 {
		if err := json.Unmarshal(bruto, &doc); err != nil {
			// NAO sobrescreve um arquivo que nao se conseguiu ler.
			//
			// Um JSON invalido pode ser um arquivo que o usuario esta editando
			// agora, ou um formato que este pacote nao entende. Nos dois casos,
			// escrever por cima destroi o que havia -- e o backup so ajuda quem
			// souber que ele existe.
			return fmt.Errorf("%s nao e um JSON valido; nada foi alterado (backup em %s%s): %w",
				caminho, caminho, SufixoDeBackup, err)
		}
	}

	servidores := map[string]json.RawMessage{}
	if bruta, ok := doc["mcpServers"]; ok && len(bruta) > 0 {
		if err := json.Unmarshal(bruta, &servidores); err != nil {
			return fmt.Errorf("mcpServers de %s nao e um objeto; nada foi alterado: %w", caminho, err)
		}
	}

	entrada, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("serializando a entrada do gobsidian: %w", err)
	}
	servidores[ChaveDoServidor] = entrada

	novosServidores, err := json.Marshal(servidores)
	if err != nil {
		return fmt.Errorf("serializando mcpServers: %w", err)
	}
	doc["mcpServers"] = novosServidores

	saida, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("serializando %s: %w", caminho, err)
	}
	// UTF-8 sem BOM: alguns leitores de JSON engasgam com ele, e o PowerShell
	// 5.1 escrevia BOM por padrao -- o comentario do instalador antigo
	// registrava isso.
	if err := os.WriteFile(caminho, append(saida, '\n'), 0o600); err != nil {
		return fmt.Errorf("gravando %s: %w", caminho, err)
	}
	return nil
}

// fazerBackup copia o arquivo antes de qualquer escrita. Falha em fazer backup
// ABORTA a fusao: editar sem rede de seguranca um arquivo que o usuario
// escreveu a mao nao vale a conveniencia.
func fazerBackup(caminho string, conteudo []byte) error {
	if err := os.WriteFile(caminho+SufixoDeBackup, conteudo, 0o600); err != nil {
		return fmt.Errorf("gravando backup de %s: %w", caminho, err)
	}
	return nil
}

// definicaoParaVSCode monta o JSON que `code --add-mcp` espera na linha de
// comando: {"name":...,"command":...,"args":[...]}.
//
// Compacto e numa linha so, porque vai como UM argumento de processo.
func definicaoParaVSCode(e Entrada) (string, error) {
	def := struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}{Name: ChaveDoServidor, Command: e.Command, Args: e.Args}
	b, err := json.Marshal(def)
	if err != nil {
		return "", fmt.Errorf("serializando a definicao para o VS Code: %w", err)
	}
	return string(b), nil
}
