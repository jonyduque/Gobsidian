package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newDoctorCmd e um placeholder ate a Task 10 implementar o diagnostico de
// verdade. Ele so precisa existir para o binario compilar e para
// "gobsidian doctor" nao ser um comando desconhecido do cobra; o
// comportamento real (checar vault, cache, permissoes) fica fora do escopo
// da Task 9. Ao contrario de serve, doctor e um comando CLI comum, entao
// escrever em stdout aqui e correto.
func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostica o ambiente (placeholder ate a Task 10)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "doctor: ainda nao implementado (ver Task 10)")
			return nil
		},
	}
}
