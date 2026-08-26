// Package vault e a unica camada que toca o sistema de arquivos do cofre.
// Todo caminho que entra aqui passa por confinamento em duas etapas antes
// de virar chamada de sistema, e e essa fronteira que garante que o
// servidor nao consegue ler nem escrever fora da raiz.
package vault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// Vault e a raiz aberta do cofre. Construir um nao valida o conteudo, so o
// caminho: um cofre vazio e legitimo, um inacessivel e erro, e distinguir
// os dois e responsabilidade de quem varre.
type Vault struct {
	root string

	// walkRoot e a raiz ja preparada para chamadas do sistema: no Windows,
	// prefixada com \\?\ quando longa. A varredura precisa dela porque e a
	// operacao com mais chance de encostar em MAX_PATH — um cofre corporativo
	// no OneDrive gasta boa parte do orcamento de 260 caracteres antes de
	// qualquer nota existir. Canonicalize e chamada contra walkRoot, nao
	// contra root, para que filepath.Rel compare formas iguais.
	walkRoot string

	// seguirSymlinks liga o comportamento anterior a 2026-08-26. Ver a opcao
	// SeguirSymlinks para a razao de o padrao ser recusar.
	seguirSymlinks bool

	// Entradas descartadas durante a varredura. Uma nota que existe no disco
	// mas nao entra no indice fica inalcancavel; sem contador, fica tambem
	// indiagnosticavel — o usuario ve uma nota sumida e nao tem por onde
	// comecar. O contador e cumulativo entre varreduras, de proposito: e um
	// sinal de saude, nao um valor por operacao.
	skipped        atomic.Int64
	skippedMu      sync.Mutex
	skippedSamples []string
}

// maxSkippedSamples limita quantos caminhos descartados sao guardados. O
// contador diz quantos; a amostra diz quais, o suficiente para diagnosticar
// sem crescer sem limite em um cofre com milhares de nomes problematicos.
const maxSkippedSamples = 50

// maxReadRangeBytes e o teto de alocacao de uma unica chamada a ReadRange.
// Sem ele, um `end` absurdo (por erro do chamador ou por um offset corrompido
// vindo do indice) aloca antes de qualquer I/O acontecer — o processo pode
// estourar memoria so por causa de dois numeros, nunca por causa do que esta
// no disco.
const maxReadRangeBytes = 64 << 20 // 64 MiB

// RecordSkip registra uma falha de leitura ou descarte durante a indexacao.
func (v *Vault) RecordSkip(abs string, cause error) {
	v.skipped.Add(1)

	v.skippedMu.Lock()
	defer v.skippedMu.Unlock()
	if len(v.skippedSamples) < maxSkippedSamples {
		v.skippedSamples = append(v.skippedSamples, fmt.Sprintf("%s: %v", abs, cause))
	}
}

// SkippedEntries devolve quantas entradas foram descartadas e uma amostra dos
// motivos. Exposto em vault_stats, e o que transforma "sumiu uma nota" em um
// diagnostico.
//
// O contador e cumulativo entre chamadas de Walk — nunca reseta sozinho.
// Isso e proposital: e um sinal de saude do cofre ao longo do tempo, nao o
// resultado de uma unica varredura. Um chamador que precisa do descarte de
// uma varredura especifica tem que ler o valor antes e depois e subtrair.
func (v *Vault) SkippedEntries() (int64, []string) {
	v.skippedMu.Lock()
	defer v.skippedMu.Unlock()

	out := make([]string, len(v.skippedSamples))
	copy(out, v.skippedSamples)
	return v.skipped.Load(), out
}

// New abre o cofre em root. Falha se o caminho nao resolve; nao verifica se
// existe nem se e legivel, porque quem diagnostica ambiente e o doctor, e
// um servidor que se recusa a subir nao consegue reportar o proprio motivo.
func New(root string, opcoes ...Opcao) (*Vault, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolvendo raiz do cofre %q: %w", root, err)
	}
	info, err := os.Stat(LongPath(abs))
	if err != nil {
		return nil, fmt.Errorf("raiz do cofre inacessivel %q: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("raiz do cofre nao e diretorio: %q", abs)
	}
	v := &Vault{root: abs, walkRoot: LongPath(abs)}
	for _, o := range opcoes {
		o(v)
	}
	return v, nil
}

// Opcao configura o cofre na construcao. Variadica para que os chamadores
// existentes de New(root) continuem valendo.
type Opcao func(*Vault)

// SeguirSymlinks liga o comportamento anterior a 2026-08-26: symlink dentro do
// cofre e seguido na varredura e na leitura.
//
// O padrao e NAO seguir, e a razao e concreta. As duas camadas de confinamento
// deste pacote sao lexicas — validateLocal e Canonicalize — e nenhuma consulta
// o disco. path.go documenta o limite e delega a "camada que abre o arquivo",
// que era um os.Open puro: um `cofre/nota.md -> C:\qualquer\coisa.txt` passava
// nas duas, entrava no indice, e note_read devolvia conteudo arbitrario do
// disco pelo canal MCP.
//
// A saida existe porque symlinkar uma pasta externa para dentro do cofre e
// workflow legitimo, suportado pelo Obsidian. Recusar sem alternativa trocaria
// um risco hipotetico — o dono do cofre e o "atacante" — por uma regressao
// certa em quem ja depende do comportamento. Quem liga a flag aceita que o
// confinamento do produto termina no symlink.
//
// A correcao ESTRUTURAL e os.Root/os.OpenRoot (Go >= 1.24), que torna o escape
// impossivel por construcao e elimina a janela TOCTOU que esta checagem deixa
// aberta. Fica como tarefa de arquitetura propria, com AD-10.
func SeguirSymlinks(seguir bool) Opcao {
	return func(v *Vault) { v.seguirSymlinks = seguir }
}

// recusaSymlink devolve erro quando o componente final do caminho e um symlink
// e a politica e nao segui-los.
//
// Usa Lstat: Stat seguiria o link, que e justamente o que se quer evitar. A
// checagem e do componente FINAL — um symlink de diretorio no meio do caminho
// nao chega aqui porque WalkDir nao desce nele.
func (v *Vault) recusaSymlink(p CanonicalPath) error {
	if v.seguirSymlinks {
		return nil
	}
	fi, err := os.Lstat(v.Abs(p))
	if err != nil {
		// Caminho inexistente ou ilegivel nao e problema desta guarda: quem
		// abrir em seguida devolve o erro real, com a mensagem certa.
		return nil
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%q e um symlink e o confinamento do cofre nao segue symlink "+
			"(use --follow-symlinks para permitir)", p)
	}
	return nil
}

// Root devolve a raiz na forma tradicional, sem o prefixo \\?\ que as
// chamadas de sistema usam internamente. E a forma que aparece em log e em
// mensagem de erro, porque e a que o usuario reconhece.
func (v *Vault) Root() string { return v.root }

// Abs devolve o caminho absoluto de um caminho canonico, ja preparado para
// chamadas do sistema.
func (v *Vault) Abs(p CanonicalPath) string {
	return LongPath(filepath.Join(v.root, filepath.FromSlash(string(p))))
}

// Open abre uma nota para leitura. Recebe CanonicalPath, nao string: o tipo
// e a prova de que o confinamento ja rodou, e nao ha caminho para chegar
// aqui sem passar por ele.
func (v *Vault) Open(p CanonicalPath) (*os.File, error) {
	if err := v.recusaSymlink(p); err != nil {
		return nil, err
	}
	f, err := os.Open(v.Abs(p))
	if err != nil {
		return nil, fmt.Errorf("abrindo %q: %w", p, err)
	}
	return f, nil
}

// ReadRange le apenas a faixa pedida. Uma secao de 2 KB em uma nota de 500 KB
// custa 2 KB — e a razao de o indice guardar offsets em vez de conteudo.
func (v *Vault) ReadRange(ctx context.Context, p CanonicalPath, start, end int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := v.recusaSymlink(p); err != nil {
		return nil, err
	}
	// start negativo precisa ser rejeitado antes de qualquer subtracao.
	// end-start so pode estourar o int64 quando start e suficientemente
	// negativo (o caso patologico e start=math.MinInt64, end=math.MaxInt64:
	// a diferenca matematica excede o range de int64 e o resultado
	// envolve para um numero negativo, que passa pelo teto abaixo sem ser
	// pego e derruba make() com um tamanho negativo). Com start >= 0
	// garantido aqui e end >= start garantido a seguir, end-start fica em
	// [0, math.MaxInt64] sempre — nunca estoura.
	if start < 0 {
		return nil, fmt.Errorf("faixa invalida em %q: start negativo (%d)", p, start)
	}
	if end < start {
		return nil, fmt.Errorf("faixa invalida em %q: %d..%d", p, start, end)
	}
	if end-start > maxReadRangeBytes {
		return nil, fmt.Errorf("faixa grande demais em %q: %d..%d excede o teto de %d bytes", p, start, end, int64(maxReadRangeBytes))
	}

	f, err := v.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, end-start)
	n, err := f.ReadAt(buf, start)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("lendo %q em %d..%d: %w", p, start, end, err)
	}
	return buf[:n], nil
}

// ReadAll le a nota inteira. Recebe ctx porque leitura de arquivo bloqueia
// de verdade — em cofre sincronizado na nuvem, indefinidamente enquanto o
// cliente hidrata o placeholder.
func (v *Vault) ReadAll(ctx context.Context, p CanonicalPath) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := v.recusaSymlink(p); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(v.Abs(p))
	if err != nil {
		return nil, fmt.Errorf("lendo %q: %w", p, err)
	}
	return data, nil
}
