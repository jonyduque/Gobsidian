# Task 90 — Reformular o RNF-30 e o analisador antes de abrir socket

**Tier: modelo forte.** O entregável é uma garantia de produto reescrita de modo
a continuar auditável. Errar aqui troca uma promessa verificável por uma
afirmação.

#### Onde encaixa
**Antes** das Tasks 91 e 92, e é bloqueante para elas. Instrumento primeiro: a
Fase 1 deste plano registra que gate que para de gatear já aconteceu duas vezes
aqui, e abrir socket sem reescrever a regra faz o `check_net` reprovar — e a
tentação seguinte é desligá-lo, o que é pior que ele não existir.

#### O que vincula esta tarefa
**Esta tarefa reabre uma decisão fechada, com autorização explícita do dono do
projeto em 2026-08-05.** O texto atual de `docs/PRD.md`:

> RNF-30 — Nenhuma requisição de rede. O código do produto não abre socket de
> saída em nenhuma circunstância.
>
> RNF-30 é uma propriedade de produto, não apenas técnica: o cofre pode conter
> material confidencial, e a garantia de que o servidor não exfiltra precisa ser
> verificável, não apenas afirmada.

A razão é de produto, não de gosto, e **a nova formulação tem de preservá-la**.

Vale também: **nenhum teto de RNF é afrouxado nesta batelada**, e **não escreva
número que você não mediu**.

#### A decisão que esta tarefa tem de acertar
A garantia passa de "nenhum socket" para **"nenhum socket que saia da máquina"**,
e continua verificável em um comando:

1. Nos nossos pacotes, `net.Dial` e `net.Listen` só com a rede **constante
   `"unix"`**. Rede vinda de variável é recusada pelo analisador — sem isso,
   `net.Dial(rede, endereco)` passa e a garantia evapora.
2. O endereço tem de ser um caminho sob o diretório de runtime do usuário. O
   analisador não consegue provar isso; o **teste** prova, e o analisador barra
   a forma que permitiria burlar.
3. `net/http`, cliente HTTP e qualquer `Dial` de `tcp` ou `udp` seguem proibidos
   nos nossos pacotes.
4. O texto do RNF-30 no PRD é reescrito com a data, a autorização e o que mudou.
   **Decisão fechada que muda vira registro, não apagamento.**

#### Armadilhas já pagas que se aplicam
- **Gate que silenciosamente parou de gatear.** O `check_net` já reportou não ter
  rodado e saiu verde. O analisador novo **tem de reprovar** um caso plantado, e
  a prova disso vai no relatório.
- **Campo com valor fixo mente sempre** — vale igual para regra de lint que só
  aparenta cobrir.

#### Verificações além dos passos
Prova de disparo, uma por regra: plantar `net.Dial("tcp", ...)`; plantar
`net.Dial(rede, ...)` com a rede numa variável; plantar uma chamada de cliente
HTTP. As três têm de reprovar. Remover as três depois e colar as seis saídas.

#### Regras de execução
`verify.ps1` verde com o analisador novo **antes** de a Task 91 começar.
Registrar no ledger antes de reportar. Escopo não encolhe em silêncio.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path tools/netcheck/netcheck.go `
  -Anchor 'if !ehConstante(arg0) || valorDe(arg0) != "unix" {' -Replacement 'if false {' `
  -Test TestNetcheckRecusaRedeVariavel -Package ./tools/netcheck/
```

**A âncora nomeia código que ainda não existe, e isso é deliberado: ela é o
contrato de nomes desta tarefa.** Se a implementação usar outro nome, a prova
não casa âncora e o `mutate.ps1` sai `2`, inconclusivo — que se lê como "não
provado", e é.

#### Contrato de relatório
As seis saídas de disparo. O diff do texto do RNF-30. A frase explícita de que a
decisão foi reaberta, por quem e quando.

**Files:** `docs/PRD.md`, `tools/netcheck/`, `scripts/check_net.ps1`, testes
**Commit:** `docs(prd): restate RNF-30 as no socket that leaves the machine`

---

