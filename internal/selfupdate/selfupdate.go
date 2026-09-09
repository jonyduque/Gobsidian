// Package selfupdate baixa uma versao nova do gobsidian e prova que ela e a
// que foi publicada.
//
// # A excecao da RNF-30 mora aqui, e so aqui
//
// Este e o UNICO pacote do produto que pode importar net/http (PRD 6.4,
// segunda reabertura, 2026-09-08, decisao D-13 do dono). Antes disso quem
// baixava era um script; com o instalador virando subcomando, o download
// passou a ser codigo Go, e a regra teve de ser reaberta em vez de contornada.
//
// O que estreita a excecao, e onde cada parte e verificada:
//
//	um pacote so             tools/netcheck (analise estatica)
//	lista fechada de hosts   tools/netcheck (literais de URL no codigo)
//	URL de runtime           ValidarHost, aqui, com teste
//	so `update` alcanca      serve, daemon e as tools nao importam este pacote
//
// A separacao entre as duas ultimas e deliberada: o analisador estatico nao
// prova para onde uma URL vinda de variavel aponta -- e a mesma limitacao que
// a regra do IPC ja registra para net.Dial. Dizer que a analise estatica cobre
// o caso de runtime seria mentira; por isso ha guarda E teste.
//
// # Nada aqui roda durante o serve
//
// Servir um cofre nunca toca a rede. Este pacote e alcancado apenas pelo
// subcomando que o usuario digitou.
package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Os hosts que este pacote pode alcancar. Lista FECHADA, e a mesma que
// tools/netcheck cobra sobre os literais deste pacote -- as duas listas
// existem porque uma e verificada em tempo de compilacao e a outra em tempo de
// execucao, e um teste aqui prova que elas nao divergiram.
const (
	HostDaAPI      = "https://api.github.com"
	HostDeDownload = "https://objects.githubusercontent.com"
	HostDoRepo     = "https://github.com"
)

// HostsPermitidos e a lista consultada em tempo de execucao.
func HostsPermitidos() []string {
	return []string{HostDaAPI, HostDeDownload, HostDoRepo}
}

// ErrHostProibido e devolvido quando uma URL aponta para fora da lista.
//
// Sentinela, e nao string: quem chama precisa distinguir "recusei por
// seguranca" de "a rede falhou", e as duas coisas pedem mensagens diferentes
// para o usuario.
var ErrHostProibido = errors.New("host fora da lista permitida")

// ErrHashDivergente e devolvido quando o arquivo baixado nao casa com a soma
// publicada. Nunca e recuperavel: um binario que nao confere nao e instalado
// (decisao D-03 do dono).
var ErrHashDivergente = errors.New("SHA-256 do arquivo baixado nao confere com o publicado")

// ValidarHost recusa qualquer URL fora da lista.
//
// E a metade que a analise estatica nao alcanca: a URL de download de um
// release vem do JSON da API, em tempo de execucao, e nenhum analisador prova
// para onde ela aponta. Sem esta guarda, uma resposta adulterada da API
// mandaria o instalador buscar o binario em qualquer lugar.
//
// Compara o PREFIXO com o host seguido de "/" -- ou o host exato. Sem o "/",
// "https://api.github.com.exemplo.invalid" passaria por ser prefixo de texto.
func ValidarHost(url string) error {
	for _, h := range HostsPermitidos() {
		if url == h || strings.HasPrefix(url, h+"/") {
			return nil
		}
	}
	return fmt.Errorf("%w: %q (permitidos: %s)", ErrHostProibido, url, strings.Join(HostsPermitidos(), ", "))
}

// Transporte busca uma URL e devolve o corpo.
//
// Interface de um metodo so para que o teste nao precise de rede: o fake
// devolve bytes de memoria, e nenhum arquivo _test.go deste pacote importa
// net. Um teste que sobe servidor para exercitar o cliente testaria o servidor
// junto, e ainda violaria a propria regra que este pacote existe para respeitar.
type Transporte interface {
	Buscar(ctx context.Context, url string) (io.ReadCloser, error)
}

// Release e o que a API devolve, reduzido ao que importa aqui.
type Release struct {
	Tag    string
	Ativos map[string]string // nome do arquivo -> URL de download
}

// respostaDaAPI espelha o JSON do GitHub. Campos que nao usamos ficam de fora
// de proposito: cada campo a mais e uma promessa a mais sobre um formato que
// nao e nosso.
type respostaDaAPI struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// UltimaVersao consulta o release mais recente publicado.
//
// repo vem no formato "dono/projeto". Ele entra na URL por concatenacao com
// uma constante, e o resultado passa por ValidarHost como qualquer outra URL:
// um repo com "../" ou com "@outrohost" nao pode desviar a chamada.
func UltimaVersao(ctx context.Context, t Transporte, repo string) (Release, error) {
	url := HostDaAPI + "/repos/" + repo + "/releases/latest"
	corpo, err := buscarValidando(ctx, t, url)
	if err != nil {
		return Release{}, err
	}
	defer func() { _ = corpo.Close() }()

	var resp respostaDaAPI
	if err := json.NewDecoder(corpo).Decode(&resp); err != nil {
		return Release{}, fmt.Errorf("lendo resposta da API de releases: %w", err)
	}
	if resp.TagName == "" {
		return Release{}, errors.New("a API devolveu um release sem tag")
	}

	r := Release{Tag: resp.TagName, Ativos: map[string]string{}}
	for _, a := range resp.Assets {
		// Ativo cuja URL nao passa na lista e DESCARTADO aqui, e nao no
		// download: assim ele nunca chega a aparecer como opcao para quem
		// escolhe o que baixar.
		if ValidarHost(a.URL) != nil {
			continue
		}
		r.Ativos[a.Name] = a.URL
	}
	return r, nil
}

// Baixar grava o ativo em destino e CONFERE o SHA-256 contra o publicado.
//
// A conferencia acontece aqui, e nao no bootstrap: e o binario VELHO que
// confere o NOVO (decisao D-03 do dono). Um binario nao verifica a si mesmo
// com credibilidade depois de ja estar rodando.
//
// Grava num temporario e so entao renomeia. Um download interrompido no meio
// nao pode deixar meio executavel no caminho de destino -- e o mesmo raciocinio
// de vault.ReplaceFile, pelo mesmo motivo.
func Baixar(ctx context.Context, t Transporte, r Release, nomeDoAtivo, destino string) error {
	url, ok := r.Ativos[nomeDoAtivo]
	if !ok {
		return fmt.Errorf("release %s nao tem o ativo %q", r.Tag, nomeDoAtivo)
	}

	esperado, err := somaPublicada(ctx, t, r, nomeDoAtivo)
	if err != nil {
		return err
	}

	corpo, err := buscarValidando(ctx, t, url)
	if err != nil {
		return err
	}
	defer func() { _ = corpo.Close() }()

	if err := os.MkdirAll(filepath.Dir(destino), 0o700); err != nil {
		return fmt.Errorf("criando diretorio de destino: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(destino), ".gobsidian-baixado-*")
	if err != nil {
		return fmt.Errorf("criando temporario de download: %w", err)
	}
	tmpNome := tmp.Name()
	defer func() { _ = os.Remove(tmpNome) }() // no-op depois do rename

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), corpo); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("gravando download: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("fechando download: %w", err)
	}

	obtido := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(obtido, esperado) {
		return fmt.Errorf("%w: esperado %s, obtido %s", ErrHashDivergente, esperado, obtido)
	}

	if err := os.Rename(tmpNome, destino); err != nil {
		return fmt.Errorf("movendo download para %s: %w", destino, err)
	}
	return nil
}

// NomeDoArquivoDeSomas e o ativo que carrega as somas de todos os outros.
const NomeDoArquivoDeSomas = "SHA256SUMS.txt"

// somaPublicada le SHA256SUMS.txt do release e devolve a soma do ativo.
//
// Release sem esse arquivo ABORTA, e nao "instala sem conferir": um binario sem
// soma publicada nao pode ser verificado, e instalar assim mesmo transformaria
// a garantia em decoracao. As releases anteriores a v1.5.1 nao o traziam, e o
// comentario em .github/workflows/release.yml registra isso.
func somaPublicada(ctx context.Context, t Transporte, r Release, nomeDoAtivo string) (string, error) {
	url, ok := r.Ativos[NomeDoArquivoDeSomas]
	if !ok {
		return "", fmt.Errorf("release %s nao publica %s; sem ele nao ha o que conferir", r.Tag, NomeDoArquivoDeSomas)
	}
	corpo, err := buscarValidando(ctx, t, url)
	if err != nil {
		return "", err
	}
	defer func() { _ = corpo.Close() }()

	dados, err := io.ReadAll(corpo)
	if err != nil {
		return "", fmt.Errorf("lendo %s: %w", NomeDoArquivoDeSomas, err)
	}
	for _, linha := range strings.Split(string(dados), "\n") {
		campos := strings.Fields(linha)
		if len(campos) != 2 {
			continue
		}
		// O formato do sha256sum(1) e "<soma>  <nome>", e o nome pode vir com
		// um "*" na frente no modo binario.
		if strings.TrimPrefix(campos[1], "*") == nomeDoAtivo {
			return campos[0], nil
		}
	}
	return "", fmt.Errorf("%s nao lista %q", NomeDoArquivoDeSomas, nomeDoAtivo)
}

// buscarValidando e o unico caminho por onde uma URL sai deste pacote.
//
// Toda chamada passa por ValidarHost ANTES do transporte, e nao depois: um
// transporte que ja discou nao pode desfazer a conexao.
func buscarValidando(ctx context.Context, t Transporte, url string) (io.ReadCloser, error) {
	if err := ValidarHost(url); err != nil {
		return nil, err
	}
	return t.Buscar(ctx, url)
}
