# Task 24: Tools de leitura e resources

## Status
Concluído (GREEN)

## Decisões Tomadas
- **TDD:** Escritos os testes `internal/mcpsrv/tools_read_test.go` simulando requisições JSON-RPC em memória usando `mcp.IOTransport`. Testado todos os 5 read tools e errors (`IsError: true`).
- **Tools:** Criados os tool handlers em `tools_read.go` delegando diretamente para `s.svc`. As descrições do jsonschema foram extraídas de `docs/TOOLS.md`. A anotação `jsonschema` com constraints de `minimum`/`maximum` etc. teve de ser limpa ou separada devido a divergências de parsing do SDK. A estrutura do `CallToolResult` devolve exatamente a estrutura que vem do `Service`, exceto quando evitamos validação em casos onde a property devolve null invés de map vazio.
- **Resources (`gobsidian://`):** Utilizou-se o método `s.mcp.AddResource` na iniciação em `registerResources` (em `resources.go`) para as top 200 notas mais recentes e `s.mcp.AddResourceTemplate` para template genérico. A formatação do output adequa-se à v1.5.0 do SDK.
- **Boot Sequence:** O índice é devidamente construído `idx.Build(ctx, v)` no `serve.go` **antes** da inicialização e exposição do MCP Server.
- **IO Transport:** O Server utiliza `mcp.IOTransport` comunicando sobre `os.Stdin` e `os.Stdout`, isolado de logs (`log.Writer()` no `stderr`).

## Próximos Passos
Prosseguir com Task 25 (Tools de escrita e permissões) ou reportar a finalização da track ao Conductor.
