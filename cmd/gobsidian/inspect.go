package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/spf13/cobra"
)

type inspectResultJSON struct {
	Path        string         `json:"path"`
	Title       string         `json:"title"`
	Size        int64          `json:"size"`
	ModTime     time.Time      `json:"mod_time"`
	Tags        []string       `json:"tags"`
	Headings    []string       `json:"headings"`
	LinksCount  int            `json:"links_count"`
	Backlinks   []string       `json:"backlinks"`
	Frontmatter map[string]any `json:"frontmatter,omitempty"`
}

func newInspectCmd() *cobra.Command {
	var flags config.Flags
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "inspect <nota>",
		Short: "Exibe metadados, links e backlinks de uma nota",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.ReadOnlySet = cmd.Flags().Changed("read-only")
			flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}

			v, err := vault.New(cfg.VaultPath)
			if err != nil {
				return err
			}

			idx := index.New()
			if err := idx.Build(cmd.Context(), v); err != nil {
				return err
			}

			canonical, err := idx.ResolvePath(args[0])
			if err != nil {
				return fmt.Errorf("resolvendo nota %q: %w", args[0], err)
			}

			n, ok := idx.Get(canonical)
			if !ok {
				return fmt.Errorf("nota %q nao encontrada no indice", canonical)
			}

			backlinksRaw := idx.Backlinks(canonical)
			var backlinks []string
			seenBL := make(map[string]bool)
			for _, bl := range backlinksRaw {
				s := string(bl.From)
				if !seenBL[s] {
					seenBL[s] = true
					backlinks = append(backlinks, s)
				}
			}

			var headings []string
			for _, h := range n.Headings {
				headings = append(headings, h.Text)
			}

			// inspect e um subcomando de CLI (nao um servidor MCP JSON-RPC),
			// portanto a escrita em stdout e feita de proposito.
			out := cmd.OutOrStdout()

			if jsonOutput {
				res := inspectResultJSON{
					Path:        string(n.Path),
					Title:       n.Title,
					Size:        n.Size,
					ModTime:     n.ModTime,
					Tags:        n.Tags,
					Headings:    headings,
					LinksCount:  len(n.Links),
					Backlinks:   backlinks,
					Frontmatter: n.Frontmatter,
				}
				b, err := json.MarshalIndent(res, "", "  ")
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(out, string(b))
				return nil
			}

			_, _ = fmt.Fprintf(out, "[OK] Inspeção da nota %q:\n", string(n.Path))
			_, _ = fmt.Fprintf(out, "[*] Título: %s\n", n.Title)
			_, _ = fmt.Fprintf(out, "[*] Tamanho: %d bytes\n", n.Size)
			if len(n.Tags) > 0 {
				_, _ = fmt.Fprintf(out, "[*] Tags (%d): %s\n", len(n.Tags), strings.Join(n.Tags, ", "))
			}
			if len(headings) > 0 {
				_, _ = fmt.Fprintf(out, "[*] Headings (%d): %s\n", len(headings), strings.Join(headings, ", "))
			}
			_, _ = fmt.Fprintf(out, "[*] Links de saída: %d\n", len(n.Links))
			if len(backlinks) > 0 {
				_, _ = fmt.Fprintf(out, "[*] Backlinks (%d): %s\n", len(backlinks), strings.Join(backlinks, ", "))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "saida estruturada em formato JSON")
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "desabilita verificacoes de escrita")
	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")

	return cmd
}
