# Task 22 Report

## O que foi implementado
- Criação das structs `Query` e `TagCount`.
- Implementação de `List` em `internal/index/query.go` cobrindo todos os filtros na sequência especificada: tags -> folder -> glob -> frontmatter.
- Implementação da lógica de matching de `frontmatter` exatamente conforme a tabela de `docs/TOOLS.md`.
- Alteração em `internal/index/index.go` para popular o mapa invertido `ix.tags` dentro de `insert()`.
- Criação de `internal/index/query_test.go` com testes focados em TDD, cobrindo todos os casos de uso obrigatórios.

## Evidência TDD
### RED (saída de teste falhando por não passar nas lógicas novas, ex. parsing e matching):
```
--- FAIL: TestQuery (0.03s)
    --- FAIL: TestQuery/Scalar_vs_List (0.00s)
    --- FAIL: TestQuery/List_vs_List_all_required (0.00s)
    --- FAIL: TestQuery/Null_for_key_presence (0.00s)
    --- FAIL: TestQuery/Dotted_key_navigation (0.00s)
    --- FAIL: TestQuery/Folder_non-recursive (0.00s)
    --- FAIL: TestQuery/Glob_* (0.00s)
    --- FAIL: TestQuery/Tags_hierarchical (0.00s)
    --- FAIL: TestQuery/Tags_any (0.00s)
    --- FAIL: TestQuery/Tags_all (0.00s)
    --- FAIL: TestQuery/Sorting_asc (0.00s)
    --- FAIL: TestQuery/Sorting_desc (0.00s)
FAIL
```
### GREEN (após implementação e ajustes de títulos e tags):
```
--- PASS: TestQuery (0.01s)
    --- PASS: TestQuery/Empty_Query_Returns_All (0.00s)
    --- PASS: TestQuery/Scalar_vs_Scalar_Accent/Case_Insensitive (0.00s)
    --- PASS: TestQuery/Scalar_vs_List (0.00s)
    --- PASS: TestQuery/List_vs_List_all_required (0.00s)
    --- PASS: TestQuery/Null_for_key_presence (0.00s)
    --- PASS: TestQuery/Missing_field (0.00s)
    --- PASS: TestQuery/Dotted_key_navigation (0.00s)
    --- PASS: TestQuery/Folder_non-recursive (0.00s)
    --- PASS: TestQuery/Folder_recursive (0.00s)
    --- PASS: TestQuery/Glob_* (0.00s)
    --- PASS: TestQuery/Tags_hierarchical (0.00s)
    --- PASS: TestQuery/Tags_any (0.00s)
    --- PASS: TestQuery/Tags_all (0.00s)
    --- PASS: TestQuery/Pagination_total_before_limit (0.00s)
    --- PASS: TestQuery/Sorting_asc (0.00s)
    --- PASS: TestQuery/Sorting_desc (0.00s)
PASS
ok      github.com/jonyd/gobsidian/internal/index   0.913s
```

## Respostas para verificações extras
- **Cada linha da tabela de frontmatter de docs/TOOLS.md tem um caso?**
  Sim, listadas linha por linha no arquivo de teste (`Scalar vs Scalar`, `Scalar vs List`, `List vs List`, `Null for key presence`, `Dotted key navigation`, `Missing field`).
- **total reflete a contagem antes do corte?**
  Sim, testado em `Pagination_total_before_limit`, o total retorna `4` antes do cut-off do limit `1`.
- **Filtro por tag pega tag hierárquica pai quando se busca o filho, e vice-versa?**
  A implementação busca por prefixo ao procurar pela tag pai. Buscar por `#A` trará notas que tenham a tag `#A` ou `#A/B`. Porém buscar por `#A/B` não traz notas que só têm `#A`. Isto reflete a funcionalidade pretendida das hierarquias de tags do Obsidian.
- **Glob com * e com ? casa o que se espera, e não atravessa separador de pasta quando não deveria?**
  Sim, a checagem com `path.Match` faz exatamente isso sem atravessar separadores `/`.
- **Ordenação por cada valor de sort, nas duas direções, é estável para valores iguais?**
  Sim, usamos `slices.SortStableFunc` e sempre fazemos fallback para ordenação por Path (`string(a.Path) < string(b.Path)`) quando os valores são iguais.
- **Uma consulta sem nenhum filtro devolve tudo, ou nada?**
  Testado em `Empty Query Returns All`, a query devolve todas as notas (4, no ambiente de testes).

## Achados e Correções
- Faltou povoar a lista invertida `ix.tags` dentro da própria indexação (`insert()`). Foi adicionada essa mecânica em `index.go`.

## Preocupações
Nenhuma.
