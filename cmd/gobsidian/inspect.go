package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jonyduque/Gobsidian/internal/boot"
	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/vault"
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
			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}

			v, err := vault.New(cfg.VaultPath, vault.SeguirSymlinks(cfg.FollowSymlinks))
			if err != nil {
				return err
			}

			log := loggerDeCLI(cmd, cfg)
			idx, _, err := boot.AbrirIndice(cmd.Context(), v, cfg, log)
			if err != nil {
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

			// Sem acento: a saida de console e ASCII puro. Um console
			// PowerShell em CP-850 renderia "Inspecao" e "Titulo" como lixo,
			// e a regra existe justamente para os comandos de diagnostico.
			con := console.New(out)
			campos := []console.Campo{
				console.Campof("titulo", "%s", n.Title),
				{Chave: "tamanho", Valor: fmt.Sprintf("%d", n.Size), Nota: "bytes"},
			}
			if len(n.Tags) > 0 {
				campos = append(campos, console.Campo{
					Chave: "tags", Valor: strings.Join(n.Tags, ", "),
					Nota: fmt.Sprintf("(%d)", len(n.Tags)),
				})
			}
			if len(headings) > 0 {
				campos = append(campos, console.Campo{
					Chave: "headings", Valor: strings.Join(headings, ", "),
					Nota: fmt.Sprintf("(%d)", len(headings)),
				})
			}
			campos = append(campos, console.Campof("links de saida", "%d", len(n.Links)))
			if len(backlinks) > 0 {
				campos = append(campos, console.Campo{
					Chave: "backlinks", Valor: strings.Join(backlinks, ", "),
					Nota: fmt.Sprintf("(%d)", len(backlinks)),
				})
			}
			con.Campos(string(n.Path), campos)
			return nil
		},
	}

	flagsDeCofre(cmd, &flags)
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "saida estruturada em formato JSON")
	flagsDeCache(cmd, &flags)

	return cmd
}
