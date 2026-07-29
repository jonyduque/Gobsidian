# Task 25: Paridade com o Obsidian - Report

## O que foi implementado
- Criação do plugin `parity-dumper` em `tools/parity-dumper` (`manifest.json`, `main.ts`, `README.md`) para extrair cache de metadados do Obsidian real.
- Montagem do corpus base em `testdata/parity/vault` (inicialmente vazio para validar o skip) e `testdata/parity/metadata.json`.
- Implementação de `internal/index/parity_test.go` com o teste principal `TestParityWithObsidian` e os ajudantes de verificação (`assertHeadingsContain`, `assertTagsContain`, `assertBlocksContain`, `assertLinksMatch`).
- Correção de erro de compilação: O código fornecido no plano para `assert*` estava usando tipos sem os referenciar adequadamente do pacote `parser`. Os pacotes `parser` foram importados e `index.ResolvedLink` foi corretamente utilizado em substituição a um `Link` cru da proposta, além de comparar `g.Kind == parser.LinkEmbed` em vez do inexistente `.IsEmbed`.

## Evidência de TDD
- **RED**: Erros de importação no arquivo base durante o setup; referências a tipos de parser incompletas no documento base.
- **GREEN**: Todos os testes em `internal/index` rodam em menos de `3.0s` com `-race`. `go vet` não relata falhas para linux, darwin, ou local.

## Verificações Extras

| Verificação | Resultado Real |
|-------------|----------------|
| Sem `testdata/parity/vault`, o teste pula com mensagem acionável? | Sim, ele checa via `os.IsNotExist` se a pasta `testdata/parity/vault` está ausente e chama `t.Skip` apontando para o README. |
| Com o corpus presente, cada `assert` reporta caminho, valor esperado e valor obtido? | Sim, todos os falhas de asserção em `assert*` fazem loop no alvo e reportam com `t.Errorf("%s: ...", path, ...)`. |
| Cada pergunta de paridade acumulada tem resposta registrada? | Para que haja documentação de paridade, os testes terão que ser rodados com as anotações do dumper. Os casos documentados no `README.md` servem de base. Como o corpus real requer Obsidian sendo executado pelo dev, e submetemos um `{}` provisório que não contém os defeitos de parseamento real ainda, isso deve ser concluído nas próximas execuções via teste e atualizado no readme. |

## Achados da Auto-Revisão e Correções no Código do Plano
1. O plano utilizou `[]Heading` e `[]Link`, mas em `index.go` os tipos corretos em `Note` são `[]parser.Heading` e `[]ResolvedLink`.
2. O plano usou uma propriedade irreal `IsEmbed` para checagem em `assertLinksMatch`. Foi substituída por `g.Kind == parser.LinkEmbed`.
3. O import no plano constava como `gobsidian/internal/vault`, enquanto o module real em `go.mod` é `github.com/jonyd/gobsidian`. Foi ajustado.

## Preocupações
Nenhuma bloqueante para este passo. O workflow precisará que o testdata seja de fato gerado e comitado posteriormente para as perguntas da Task 12 ganharem fechamento em código (via ajustes em parser de frontmatter ou tolerância a espaçamentos).
