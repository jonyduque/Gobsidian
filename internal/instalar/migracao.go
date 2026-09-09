package instalar

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/search"
)

// MigracaoDeChave descreve um diretorio de cache cujo NOME nao e mais a chave
// que config.VaultKey produz para o cofre dele.
//
// Feito diz se a migracao aconteceu; quando nao, Motivo diz por que. Um
// relatorio que so lista o que deu certo faz uma migracao que falhou em tudo
// parecer uma que nao tinha o que fazer -- e o mesmo achado que fez o
// Relatorio da limpeza ganhar NaoRemovidos.
type MigracaoDeChave struct {
	De     string
	Para   string
	Cofre  string
	Feito  bool
	Motivo string
}

// MigrarChaves renomeia diretorios de cache cuja chave ficou para tras.
//
// # Por que isto existe
//
// config.VaultKey nomeia o diretorio de cache e o caminho do socket. Em
// 2026-09-09 a conta dela deixou de usar strings.ToLower -- tabela da stdlib,
// que a toolchain move sem avisar -- e passou a usar cases.Lower do x/text,
// modulo fixado. Ver o comentario de caixaEstavel para o defeito inteiro.
//
// Medido antes de trocar: para caminho latino, inclusive acentuado, as duas
// contas dao o MESMO hash. Diferem so onde o x/text aplica casing especial que
// o ToLower simples nao aplica -- sigma final grego, I com ponto turco. Ou
// seja: esta migracao e um nao-evento para quase todo mundo, e existe para
// quem nao esta nesse "quase".
//
// # Por que renomear, e nao apagar e reconstruir
//
// A decisao D-05 do dono e que cache de cofre EXISTENTE nunca e tocado pela
// limpeza, e reconstruir custou 3021 ms no cofre de referencia. Renomear
// preserva o cache inteiro e nao contradiz D-05: nada e removido.
//
// # Por que isto NAO roda na partida do servidor
//
// E I/O, e a invariante de cmd/gobsidian/serve.go diz que nada roda antes de
// os mecanismos de encerramento estarem armados (scripts/check_partida.ps1
// cobra). Alem disso, renomear diretorio de cache enquanto alguem o mapeia com
// mmap e pedir problema. Roda de onde a limpeza ja roda: `install`, `update` e
// `doctor` -- os dois primeiros sob a trava global, com nenhum processo do
// produto no ar.
//
// A prova de que um diretorio esta sob chave superada e mecanica, e nao
// depende de conhecer a conta ANTIGA: o cabecalho do cache guarda o caminho do
// cofre, e a chave que ele deveria ter e config.VaultKey desse caminho. Serve
// para esta mudanca e para qualquer outra que venha.
func MigrarChaves(cacheRaiz string, aplicar bool) ([]MigracaoDeChave, error) {
	entradas, err := os.ReadDir(cacheRaiz)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("lendo raiz do cache %s: %w", cacheRaiz, err)
	}

	// Os nomes que ja existem, para nunca renomear por cima de um cache vivo.
	ocupado := map[string]bool{}
	for _, e := range entradas {
		if e.IsDir() {
			ocupado[e.Name()] = true
		}
	}

	var saida []MigracaoDeChave
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		h, err := search.LerCabecalhoDoCache(filepath.Join(cacheRaiz, e.Name()))
		if err != nil || h.VaultPath == "" {
			// Sem cabecalho legivel nao ha prova de qual cofre e este
			// diretorio. Intocado, como na limpeza.
			continue
		}
		esperada := config.VaultKey(h.VaultPath)
		if esperada == e.Name() {
			continue
		}

		m := MigracaoDeChave{De: e.Name(), Para: esperada, Cofre: h.VaultPath}

		if _, err := os.Stat(h.VaultPath); err != nil {
			// Cofre sumido: isto e trabalho da limpeza, sob a regra dela.
			// Renomear lixo so o deixa com nome novo.
			m.Motivo = "o cofre do cabecalho nao existe mais; assunto da limpeza"
			saida = append(saida, m)
			continue
		}
		if ocupado[esperada] {
			// Ja existe cache sob a chave nova. Ele e o valido; este e um
			// resto. Nao apagamos nada (D-05) e nao sobrescrevemos nada.
			m.Motivo = "ja existe diretorio sob a chave nova; nada foi tocado"
			saida = append(saida, m)
			continue
		}
		if !aplicar {
			m.Motivo = "simulacao"
			saida = append(saida, m)
			continue
		}
		if err := os.Rename(filepath.Join(cacheRaiz, e.Name()), filepath.Join(cacheRaiz, esperada)); err != nil {
			m.Motivo = err.Error()
			saida = append(saida, m)
			continue
		}
		ocupado[esperada] = true
		m.Feito = true
		saida = append(saida, m)
	}

	sort.Slice(saida, func(i, j int) bool { return saida[i].De < saida[j].De })
	return saida, nil
}
