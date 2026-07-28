package index

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
	"golang.org/x/sync/errgroup"
)

type parsed struct {
	entry vault.Entry
	note  *parser.ParsedNote
	hash  uint64
	eol   vault.EOLStyle
	bom   bool
}

// Build varre o cofre e constroi o indice do zero.
//
// A varredura enfileira caminhos; um worker pool le e parseia em paralelo; um
// unico coletor popula o indice. O coletor ser unico e o que dispensa lock no
// caminho quente e o que torna o resultado independente da ordem de conclusao.
func (ix *Index) Build(ctx context.Context, v *vault.Vault) error {
	entries := make(chan vault.Entry, 256)
	results := make(chan parsed, 256)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		defer close(entries)
		return v.Walk(gctx, func(e vault.Entry) error {
			select {
			case entries <- e:
				return nil
			case <-gctx.Done():
				return gctx.Err()
			}
		})
	})

	workers := runtime.NumCPU()
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		g.Go(func() error {
			defer wg.Done()
			for e := range entries {
				if err := gctx.Err(); err != nil {
					return err
				}

				// Anexos e placeholders de nuvem nao sao lidos. Ler um
				// placeholder dispararia download sincrono, e indexar o cofre
				// inteiro assim trava por minutos.
				if !e.IsNote || e.CloudOnly {
					select {
					case results <- parsed{entry: e}:
					case <-gctx.Done():
						return gctx.Err()
					}
					continue
				}

				data, err := v.ReadAll(gctx, e.Path)
				if err != nil {
					// Um arquivo ilegivel nao derruba a indexacao inteira.
					continue
				}

				body, hadBOM := vault.StripBOM(data)
				note, err := parser.Parse(body)
				if err != nil {
					continue
				}
				if hadBOM {
					note.ShiftOffsets(int64(vault.BOMLen))
				}

				select {
				case results <- parsed{
					entry: e,
					note:  note,
					hash:  xxhash.Sum64(data),
					eol:   vault.DetectEOL(data),
					bom:   hadBOM,
				}:
				case <-gctx.Done():
					return gctx.Err()
				}
			}
			return nil
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	g.Go(func() error {
		for r := range results {
			ix.insert(r)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("construindo indice: %w", err)
	}

	// As tres passadas seguintes dependem do conjunto completo e por isso
	// acontecem depois, nao durante.
	ix.buildAliasMap()
	ix.resolveAllLinks()
	ix.buildBacklinks()

	return nil
}
