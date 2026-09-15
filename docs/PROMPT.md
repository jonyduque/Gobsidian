# Instruções que o servidor manda ao modelo

O host MCP recebe, de cada tool, **só `type` e `description`** — limitação
registrada em [`TOOLS.md`](TOOLS.md) na seção "Schemas servidos". Nenhum `enum`,
`minimum`, `maximum` ou `default` chega ao modelo. Ou seja: os conjuntos
fechados existem, o servidor os cobra com `INVALID_ARGUMENT` — e o modelo não
tem como saber quais são.

É esse buraco que as instruções tapam. Sem elas o modelo acerta os nomes das
tools e erra os valores, e descobre cada conjunto fechado por tentativa.

**Este arquivo não é normativo.** O contrato de cada tool é o de
[`TOOLS.md`](TOOLS.md); as instruções só carregam o que o host não consegue
transmitir.

---

## Desde 2026-09-15 elas chegam sozinhas

O texto vai no campo `instructions` do resultado do `initialize`, que o host
pode pôr no contexto do modelo. **Não é mais preciso colar nada.** Até essa data
o servidor mandava o campo vazio, e este arquivo trazia um bloco para o usuário
colar à mão em cada host — quem colou pode apagar a cópia, que agora duplica o
que o servidor já manda.

A única cópia do texto é
[`internal/mcpsrv/instrucoes.txt`](../internal/mcpsrv/instrucoes.txt), embutida
no binário. `scripts/check_prompt.ps1` prova, a cada `verify.ps1`, que os
conjuntos fechados citados nela são os mesmos que `internal/service` cobra.

Por sessão, o servidor acrescenta duas coisas antes do texto
(`internal/mcpsrv/instrucoes.go`):

- `Cofre: <nome da pasta>` — quem tem vários cofres tem vários servidores
  gobsidian no mesmo host, e o modelo precisa saber qual é qual;
- o aviso de somente-leitura, **só** quando `--read-only` está ligado. A frase
  fixa antiga ("o cofre pode estar em modo somente-leitura") valia para todo
  cofre e não dizia nada sobre este.

**Se o seu host ignora `instructions`:** copie o conteúdo de `instrucoes.txt`
para o campo de instruções que ele oferecer. Não foi medido quais hosts mostram
o campo ao modelo.

| Host | Onde colar, se precisar |
|---|---|
| Claude Code | `CLAUDE.md` do projeto, ou `~/.claude/CLAUDE.md` |
| Claude Desktop | Instruções do Project |
| Codex / Copilot | `AGENTS.md` |
| Outros | O campo de instruções de sistema que o host oferecer |

---

## Por que cada bloco está aí

**Caminhos e casing.** O erro de ambiguidade é deliberado: `docs/TOOLS.md`
registra que o servidor **nunca** escolhe entre dois candidatos. Um modelo que
não sabe disso trata a falha como bug e repete a mesma chamada.

**`expected_hash`.** É a única defesa contra escrita sobre edição concorrente do
Obsidian, e nada no schema a anuncia. Sem esta linha o modelo escreve sem hash
porque o parâmetro parece opcional — e ele é opcional, e é assim que se perde
texto do usuário.

**Conjuntos fechados.** A razão de as instruções existirem. São cobrados em
`internal/service` por `ValidarEnum`, e `check_prompt.ps1` compara a lista das
instruções com aquelas chamadas.

**Clamp contra recusa.** `limit: 5000` não falha: volta 500. Um modelo que
assume falha-ou-obedece conclui que o cofre tem 500 notas.

**Anexos e somente-nuvem.** Duas regras de produto que só aparecem como ausência
de resultado. Ver `docs/WINDOWS.md` para o mecanismo do somente-nuvem.

**Custo.** O texto tem cerca de 75 linhas e entra em toda sessão de todo host
que o mostre ao modelo. Não medido em tokens.
