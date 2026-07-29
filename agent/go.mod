// Este diretorio guarda assets de plugins de skills, nao codigo deste projeto.
// Alguns sao .go de exemplo com imports que nao resolvem (viper, fatih/color,
// github.com/you/myapp) porque sao trechos ilustrativos, nao codigo compilavel.
//
// Sem este go.mod eles pertencem ao modulo principal, e entao:
//   - `go build ./...`, `go test ./...`, `go vet ./...`, `gofmt -l .` e
//     check_net reprovam — as SETE etapas do verify.ps1 caiam por causa deles;
//   - `gopls` reporta os mesmos erros como diagnosticos do workspace, e um
//     agente que confia no LSP tenta "consertar" adicionando as dependencias,
//     que e exatamente o que a regra de nunca rodar `go mod tidy` protege.
//
// Um diretorio com go.mod proprio sai do grafo de pacotes do modulo pai. E o
// mecanismo do Go para isto; nao ha flag de exclusao equivalente.
//
// Se um plugin apagar este arquivo numa atualizacao, o sintoma sao as sete
// etapas do verify.ps1 reprovando de novo com erros em agent/. Recrie-o.
module gobsidian-agent-assets

go 1.25.0
