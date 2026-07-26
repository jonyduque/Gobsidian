package vault

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type Vault struct {
	root string

	// walkRoot e a raiz ja preparada para chamadas do sistema: no Windows,
	// prefixada com \\?\ quando longa. A varredura precisa dela porque e a
	// operacao com mais chance de encostar em MAX_PATH — um cofre corporativo
	// no OneDrive gasta boa parte do orcamento de 260 caracteres antes de
	// qualquer nota existir. Canonicalize e chamada contra walkRoot, nao
	// contra root, para que filepath.Rel compare formas iguais.
	walkRoot string

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

func (v *Vault) recordSkip(abs string, cause error) {
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

func New(root string) (*Vault, error) {
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
	return &Vault{root: abs, walkRoot: LongPath(abs)}, nil
}

func (v *Vault) Root() string { return v.root }

// Abs devolve o caminho absoluto de um caminho canonico, ja preparado para
// chamadas do sistema.
func (v *Vault) Abs(p CanonicalPath) string {
	return LongPath(filepath.Join(v.root, filepath.FromSlash(string(p))))
}

func (v *Vault) Open(p CanonicalPath) (*os.File, error) {
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
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("lendo %q em %d..%d: %w", p, start, end, err)
	}
	return buf[:n], nil
}

func (v *Vault) ReadAll(ctx context.Context, p CanonicalPath) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(v.Abs(p))
	if err != nil {
		return nil, fmt.Errorf("lendo %q: %w", p, err)
	}
	return data, nil
}
