### Task 70: `gen_vault.ps1` — cofre sintético determinístico de 5.000 notas

**Onde encaixa.** H2 do M6, e pré-requisito das Tasks 71 e 73. `docs/OPERACAO.md` registra hoje, em "O que falta": *"Rodar contra o cofre de referência do PRD: 5.000 notas, 50 MB. Até lá, RNF-01 e RNF-07 em escala de 5.000 notas seguem não validados — medidos em escala pequena, o que é diferente de medidos."*

**A decisão fechada que te vincula (nº 1 do M6):** o cofre é **gerado, não versionado**. 50 MB não entram no repositório. O gerador produz o mesmo cofre byte a byte a partir de uma semente fixa, e é a semente que é versionada.

**Entregável.** `scripts/gen_vault.ps1`, com parâmetros `-Out <dir>`, `-Notes <n>` (padrão 5000) e `-Seed <int>` (padrão fixo e documentado no cabeçalho).

O cofre precisa **parecer com um cofre real**, senão mede outra coisa:

- Profundidade de pastas variada, não tudo na raiz.
- Frontmatter com tags, e as tags com distribuição desigual — uma dúzia de tags cobrindo a maioria das notas e uma cauda longa.
- Wikilinks entre notas, **incluindo alguns quebrados** e alguns com âncora. Um cofre sem link quebrado não exercita `broken_links`.
- Aliases em parte das notas.
- Alguns anexos (`.png`, `.pdf`) — o filtro de anexo por nome precisa ter o que filtrar.
- Notas com BOM e notas sem, CRLF e LF misturados. `vault.StripBOM` e `writer.DetectEOL` existem porque isso acontece.
- Acentuação portuguesa no corpo e nos nomes de arquivo.

**Determinismo é o critério, e é verificável.** Gere duas vezes em diretórios diferentes com a mesma semente e compare por hash:

```powershell
pwsh -File scripts/gen_vault.ps1 -Out $env:TEMP\v1 -Seed 42
pwsh -File scripts/gen_vault.ps1 -Out $env:TEMP\v2 -Seed 42
# hash de cada arquivo relativo, ordenado, e um hash do conjunto
```

#### Verificações além dos passos

- Os dois cofres têm de bater **arquivo a arquivo, por SHA-256**, e não só na contagem. Contagem igual com conteúdo diferente é o modo de falha esperado de um gerador com aleatoriedade não semeada.
- Semente **diferente** tem de produzir cofre diferente. Sem esta segunda checagem, um gerador que ignora a semente por completo passa na primeira.
- Registre o tamanho em MB e a contagem real de notas, anexos, links e links quebrados. Estes números vão para a Task 71.
- Saída de console em ASCII puro: `[OK]`, `[!]`, `[i]`, `[...]`. Console PowerShell em CP-850 renderiza o resto como lixo.

#### Prova de mutação obrigatória

Neutralize a semente — faça o gerador ignorar `-Seed` e usar aleatoriedade do sistema — e confirme que a comparação dos dois cofres **reprova**. Cole a saída. Se ela não reprovar, o teste de determinismo não está verificando determinismo.

`scripts/mutate.ps1` **não serve aqui**: ele roda teste Go com `-Test` e `-Package`, e o alvo desta prova não é teste Go. A prova é a remoção descrita acima, com a saída colada — mesma disciplina, ferramenta diferente.

#### Regras de execução

Idênticas às da Task 69: `verify.ps1` verde sem `| tail`; nunca `git checkout/restore/stash/clean/reset`; nunca `go mod tidy`; ASCII puro no console; Conventional Commits em inglês.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-70-report.md`: o comando de geração e sua saída; a comparação por hash dos dois cofres com a mesma semente, colada; a comparação com sementes diferentes; os números do cofre (notas, anexos, MB, links, links quebrados); a prova de mutação da semente; e o que ficou de fora.

Responda com no máximo 15 linhas.

**Files:** Create `scripts/gen_vault.ps1`; modify `docs/OPERACAO.md`
**Commit:** `tooling: deterministic 5000-note synthetic vault generator`

---

