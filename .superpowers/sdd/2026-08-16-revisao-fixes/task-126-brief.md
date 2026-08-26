# Task 126 — `EnsureStarted` decide por handshake, nunca por errno

**Tier: modelo principal.** Concorrência, orçamento e uma correção de premissa
do próprio plano.

#### Onde encaixa
Depois da 124 e da 125. Fecha, junto com o **item 11** da Fase 4, a cadeia que
produziu o incidente de campo.

#### A evidência (2026-08-26, máquina do dono)
Nas três sessões MCP, em **toda** partida, ao longo de dias:

```
time=2026-08-25T18:26:36 msg="socket do daemon indisponivel; tentando iniciar o daemon"
     err="... connect: An invalid argument was supplied."
time=2026-08-25T18:26:46 msg="nao foi possivel iniciar o daemon; servindo em processo"
     err="socket do daemon nao respondeu em 10s: ..."
time=2026-08-25T18:26:48 msg="servidor pronto" vault=...\Estudo notes=2557 index_origin=cache
```

Dez segundos de pena por sessão, sempre terminando em `servindo em processo`, e
**nenhuma linha `"daemon iniciado"`** nos logs de runtime naquele dia: o daemon
não morreu, não nasceu. Depois de remover os `.sock` órfãos, a mesma partida no
mesmo cofre com o mesmo binário passou a decidir em **559 ms**, com
`conectado ao daemon recem-iniciado via socket`.

#### Correção de premissa — leia antes de codar
O item 11 da Fase 4 dizia: *"discar antes de desvincular — `ECONNREFUSED`
significa órfão e libera o unlink, sucesso significa abortar."*

**A primeira metade está errada como critério.** Medido em 2026-08-26 com
`net.Dial("unix", …)` no Windows: `ECONNREFUSED` (`10061`) é o que devolvem
**arquivo comum, socket órfão de dono morto à força, e caminho inexistente** —
os três. E o erro que apareceu em produção na máquina do dono foi `10022`, que
nenhum desses três reproduz. Classificar por errno decide certo nos casos que já
conhecemos e erra em silêncio no que apareceu de verdade.

**O critério é comportamental: só um handshake bem-sucedido prova daemon vivo.**
Qualquer falha de dial — refused, invalid argument, timeout — significa "não
está servindo", e libera o unlink.

#### O que entra
1. **Espera escalonada no lugar do probe único.** N tentativas de dial dentro do
   orçamento total (ex.: 250 ms × 40 = 10 s), falhando só no fim. Hoje o
   orçamento inteiro é gasto num `select` só, então o chegante tardio nunca vê o
   daemon que subiu 300 ms depois dele.
2. **A decisão é o handshake.** `EnsureStarted` só declara sucesso quando
   `DialAndHandshake` completa. Conectar não basta: um daemon com o `acceptLoop`
   morto (item 10) aceita no backlog do SO e nunca responde.
3. **O errno vai para o log** em toda tentativa falha (produzido na Task 124(c);
   aqui, consumir).
4. **`cmd/gobsidian/daemon.go` participa do mesmo lock** (`daemon/adquirirLock`)
   antes do `Listen`, com o segundo dial idempotente que a ponte já faz.

#### O que prova esta tarefa
- **RED 1:** listener que aceita a conexão e **nunca responde** ao handshake ⇒
  `EnsureStarted` não pode declarar sucesso. Falha hoje.
- **RED 2:** socket que só passa a responder após 2 s ⇒ com orçamento de 10 s, a
  ponte **conecta**, em vez de cair para o modo em processo. Falha hoje.
- **RED 3 (tempestade):** ≥10 pontes simultâneas sobre cofre frio ⇒ exatamente
  **um** daemon, e todas as sessões listam tools dentro do prazo.

Provas de mutação:
1. Trocar o critério de handshake por "dial conectou" ⇒ RED 1 reprova.
2. Remover a espera escalonada (voltar ao probe único) ⇒ RED 2 e RED 3 reprovam.

**Cuidado com o harness:** `StreamReader.Peek()` bloqueia apesar do nome, e já
deixou um ciclo 15h44m parado. Use `ReadLineAsync` com `Wait` limitado. Gate que
pode travar indefinidamente vira gate que se aprende a pular.

#### Verificações
Além dos passos:
1. O fallback em processo **continua obrigatório** nos três pontos (decisão 2 da
   Task 91, mantida pela 92). Esta tarefa muda quando ele dispara, nunca se ele
   existe. Um daemon quebrado não pode transformar a ferramenta em nada.
2. A goroutine vigia de `ln.Close()` (`daemon.go:140-143`) fica como está.
3. Meça o tempo de decisão da ponte antes e depois, no mesmo cofre e com cache
   quente. Número não medido não entra no relatório — escreva "não medido".
4. Rode `pwsh -File scripts/test_orphans.ps1 -Cycles 100` (os quatro cenários, o
   padrão) e confira que `daemon-idle` continua reprovando por `reason=` errado.
5. Não rode o gate de órfãos concorrente com a medição de tempo: um mata os
   processos do outro e produz falso verde.

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde, contagem de passos colada.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`.
- Nunca `go mod tidy`.
- Matar processo **sempre por PID que você mesmo lançou**, nunca por nome:
  `Stop-Process -Name gobsidian` já matou a sessão real do dono.
- Asserção de tempo atrás de build tag `//go:build race`, em arquivo separado.
- `pipe engole código de saída`: `cmd | tail` devolve o status do `tail`.
  Redirecione para arquivo e leia o `$?` do comando.

#### Comando de mutação
```bash
pwsh -File scripts/mutate.ps1 -Path internal/daemon/lock.go `
  -Anchor '<a linha que exige handshake bem-sucedido>' `
  -Replacement '<a versao que aceita dial conectado>' `
  -Test TestEnsureStartedNaoAceitaDaemonQueNaoResponde -Package ./internal/daemon/
```

Copie as duas âncoras **do arquivo** depois de escrever o código. A segunda
mutação (remover a espera escalonada) usa a mesma ferramenta, com a âncora do
laço de tentativas.

**Files:** `internal/daemon/lock.go`, `internal/ipc/ipc.go`,
`cmd/gobsidian/daemon.go`, `cmd/gobsidian/ponte.go`, os `_test.go`
correspondentes, e `docs/OPERACAO.md` (limites conhecidos: a corrida residual).

#### Contrato de relatório
Os três RED com saída falhando e passando; as duas mutações com `EXIT=0`; o
tempo medido de decisão da ponte **antes e depois** (a referência é 10 s →
559 ms, medida no cofre Estudo, 2.557 notas, cache quente); `verify.ps1`.

---

## Identidade do cofre — verificado, NÃO virou tarefa

Estava planejado como Task 127 e foi **retirado depois de conferir o código**,
em 2026-08-26. Fica registrado para não ser re-proposto.

**O problema observado.** `config.VaultKey` é
`xxhash.Sum64String(strings.ToLower(vaultPath))`, sem normalização Unicode. Na
máquina do dono isso produziu, para o mesmo cofre pretendido, duas instâncias
completas — socket, cache e daemon próprios para cada grafia:

| Grafia | VaultKey | Existe no disco? |
|---|---|---|
| `...\Obsidian\Jurisprudência` | `d34d3da9c925ef62` | sim |
| `...\Obsidian\Jurisprudencia` | `4568ecbd07c39faa` | **não** |
| `...\Obsidian\Revisão` | `1f213394ace393eb` | sim |
| `...\Obsidian\Revis<U+FFFD>o` | `7a43b2b161338f9a` | **não** |

As duas grafias inexistentes vieram do config do host — uma com acento removido,
outra com caractere de substituição, resíduo de round-trip de encoding.

**Por que não virou tarefa.** A correção planejada era "recusar alto caminho que
não existe". **Isso já está implementado**: `vault.New`
(`internal/vault/vault.go:90-95`) devolve `raiz do cofre inacessivel %q` para
caminho ausente e `raiz do cofre nao e diretorio: %q` para arquivo. O erro
existe, é específico, e nomeia o caminho.

O que faltava não era a validação — era **o erro chegar a alguém**. Ele é
devolvido e descartado sem log, e é por isso que o daemon morria mudo. Isso é a
Task 124(a)/(b), e a grafia real do disco aparece na checagem `grafia_do_cofre`
da Task 125. Uma tarefa própria duplicaria as duas.

**Decisão do dono, registrada:** **não** normalizar Unicode em `VaultKey`. NFC
resolveria acento composto × pré-composto e **não** resolveria acento ausente,
que é o caso que de fato ocorreu, ao custo de invalidar todo cache existente.
A chave é sensível a grafia por construção, e isso é aceitável dado que
`vault.New` recusa e o `doctor` mostra. Quem quiser reabrir precisa de caso novo
em que a divergência ocorra com **duas grafias que existem no disco**.

---

