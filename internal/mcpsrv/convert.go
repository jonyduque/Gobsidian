package mcpsrv

import (
	"fmt"

	"github.com/jonyd/gobsidian/internal/service"
)

const codeInternal = service.CodeInternal

// toolErr traduz um erro de dominio em um erro Go cujo texto comeca com o
// codigo legivel por maquina. Devolvido como error (nao como resultado), para
// que o SDK monte o CallToolResult de erro sozinho e nao serialize o valor
// zero de Out em StructuredContent — o SDK so pula StructuredContent quando o
// handler devolve um error Go nao-nulo. Sintetizar o resultado manualmente,
// como a versao anterior fazia, deixava structuredContent com um objeto
// zerado indistinguivel de um cofre legitimamente vazio.
func toolErr(err error) error {
	return fmt.Errorf("%s: %s", service.CodeOf(err), err.Error())
}
