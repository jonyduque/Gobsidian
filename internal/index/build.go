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
	// links ja vem montado do worker, com o contexto recortado. O corpo NAO
	// viaja nesta struct de proposito: ela atravessa um canal com o cofre
	// inteiro em voo, e carregar os bytes de cada nota ate o coletor
	// multiplicaria o pico de memoria pelo tamanho do cofre. O contexto e o
	// unico pedaco do corpo que o indice guarda, e ele e recortado onde os
	// bytes ja estao na mao.
	links []ResolvedLink
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
				// inteiro assim trava por minutos. Quem decide e classificar,
				// a mesma funcao que insert e Replace consultam.
				if !classificar(e).precisaLer() {
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
					v.RecordSkip(string(e.Path), err)
					continue
				}

				body, hadBOM := vault.StripBOM(data)
				note := parser.Parse(body)
				if hadBOM {
					note.ShiftOffsets(int64(vault.BOMLen))
				}

				select {
				case results <- parsed{
					entry: e,
					note:  note,
					links: montarLinks(data, note.Links),
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
	ix.resolveAllLinks()
	ix.buildBacklinks()

	return nil
}
