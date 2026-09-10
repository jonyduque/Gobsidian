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
	"sort"
	"strings"
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

// Entrada e o que se escreve sob mcpServers.<chave>.
type Entrada struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// EntradaNomeada e uma Entrada com a chave sob a qual ela aparece.
//
// Existe desde 2026-09-09, quando a instalacao passou a configurar N cofres em
// vez de um. Um cofre so continua saindo sob ChaveDoServidor -- ninguem que ja
// usa o produto tem o config mexido de graca.
type EntradaNomeada struct {
	Chave string
	Entrada
}

// PrefixoDeCofre e o comeco de toda chave de cofre nomeado.
//
// A chave INTEIRA e montada em internal/instalar (EntradasParaCofres), e nao
// aqui: montar o nome exige tirar acento, e essa conta ja existe em
// text.RemoveAccents. `hosts` e folha e nao ganha import -- ver o grafo no
// CLAUDE.md.
const PrefixoDeCofre = ChaveDoServidor + "-"

// EhNossa diz se uma chave de servidor MCP foi escrita por este programa.
//
// E o que permite podar: sem ela, reconfigurar de tres cofres para um deixaria
// duas entradas mortas apontando para um cofre que o usuario nao escolheu mais.
// Nunca casa a entrada de outro servidor -- so ChaveDoServidor e o prefixo
// dela seguido de hifen.
func EhNossa(chave string) bool {
	return chave == ChaveDoServidor || strings.HasPrefix(chave, PrefixoDeCofre)
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
	return FundirVarias(caminho, []EntradaNomeada{{Chave: ChaveDoServidor, Entrada: e}})
}

// FundirVarias escreve N entradas e PODA as nossas que sobraram.
//
// Podar e o que faz a escolha do usuario valer: quem configurou tres cofres e
// depois escolheu um precisa terminar com um, e nao com um novo mais dois
// mortos. So entrada NOSSA e removida (EhNossa) -- servidor MCP de outro
// produto atravessa a fusao como qualquer outro campo.
//
// Lista VAZIA e uma escolha valida: significa "nao configure nada", e entao o
// que este pacote escreveu antes sai e mais nada entra.
func FundirVarias(caminho string, entradas []EntradaNomeada) error {
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

	// Poda antes de escrever: uma chave que era nossa e nao esta na lista nova
	// some, e uma que continua e simplesmente sobrescrita logo abaixo.
	for chave := range servidores {
		if EhNossa(chave) {
			delete(servidores, chave)
		}
	}
	for _, en := range entradas {
		bruta, err := json.Marshal(en.Entrada)
		if err != nil {
			return fmt.Errorf("serializando a entrada %q: %w", en.Chave, err)
		}
		servidores[en.Chave] = bruta
	}

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
func definicaoParaVSCode(en EntradaNomeada) (string, error) {
	def := struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}{Name: en.Chave, Command: en.Command, Args: en.Args}
	b, err := json.Marshal(def)
	if err != nil {
		return "", fmt.Errorf("serializando a definicao para o VS Code: %w", err)
	}
	return string(b), nil
}

// LerEntradas devolve as entradas NOSSAS que ja estao num config de host.
//
// E o que responde "o que ja esta configurado?" antes de perguntar qualquer
// coisa ao usuario. Sem isso a instalacao so sabia oferecer uma lista de
// cofres do Obsidian, como se nada existisse -- e quem ja tinha configurado
// dois cofres tinha de reconfigurar os dois para nao perder um.
//
// Arquivo ausente devolve nada, sem erro: nao ter configuracao e um estado
// normal. JSON invalido TAMBEM devolve nada, e aqui isso e deliberado -- ler e
// so para perguntar melhor, e um config que este pacote nao entende ja e
// recusado na hora de escrever (ver Fundir).
func LerEntradas(caminho string) ([]EntradaNomeada, error) {
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("lendo %s: %w", caminho, err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &doc); err != nil {
		return nil, nil
	}
	bruta, ok := doc["mcpServers"]
	if !ok || len(bruta) == 0 {
		return nil, nil
	}
	var servidores map[string]json.RawMessage
	if err := json.Unmarshal(bruta, &servidores); err != nil {
		return nil, nil
	}

	var saida []EntradaNomeada
	for chave, valor := range servidores {
		if !EhNossa(chave) {
			continue
		}
		var e Entrada
		if err := json.Unmarshal(valor, &e); err != nil {
			// Entrada nossa que nao decodifica: conta como presente, porque
			// ela EXISTE e vai ser podada. O caminho do cofre fica vazio, e
			// quem mostra a lista trata isso.
			saida = append(saida, EntradaNomeada{Chave: chave})
			continue
		}
		saida = append(saida, EntradaNomeada{Chave: chave, Entrada: e})
	}
	sort.Slice(saida, func(i, j int) bool { return saida[i].Chave < saida[j].Chave })
	return saida, nil
}

// CofreDaEntrada extrai o caminho do cofre de uma entrada, lendo o argumento
// que vem depois de --vault.
//
// Devolve "" quando a entrada nao tem --vault: pode ser de uma versao antiga
// ou editada a mao, e inventar um caminho seria pior que admitir que nao se
// sabe.
func CofreDaEntrada(e Entrada) string {
	for i, a := range e.Args {
		if a == "--vault" && i+1 < len(e.Args) {
			return e.Args[i+1]
		}
		if caminho, achou := strings.CutPrefix(a, "--vault="); achou {
			return caminho
		}
	}
	return ""
}
