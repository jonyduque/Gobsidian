# Task 93 — Medição multi-instância, documentação e fechamento da Parte II

**Tier: modelo forte.** O entregável são números e uma decisão de manutenção, e
o modo de falha de um modelo barato pedido a "escrever relatório com evidência"
é fabricá-la.

#### Onde encaixa
Fechamento da Parte II. Não envia código.

#### O que vincula esta tarefa
- **Não escreva número que você não mediu.** Alvo não medido apresentado como
  resultado é ficção com aparência de tabela. Onde não mediu, escreva
  "não medido".
- **Confira todo SHA que você escrever no ledger.** A Task 31 foi registrada em
  `14210ee`, que não existe no repositório.
- **Escopo não encolhe em silêncio.**

#### O que entregar
- Tabela de RSS para **uma, três e cinco** sessões simultâneas no cofre real, em
  três configurações: hoje, com a Task 88, e com o daemon.
- `docs/ARCHITECTURE.md` ganha a seção do daemon e do transporte, com a medição
  de AF_UNIX contra named pipe e a razão da escolha.
- `docs/PRD.md` com o RNF-30 já reformulado pela Task 90 — conferir que ficou.
- `README.md`: como desligar o daemon, e o que acontece quando ele não sobe.
- Ledger em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`.
- **Uma recomendação explícita:** o daemon compensa? Se o ganho agregado não
  justificar um processo de longa vida a mais para manter, dizer isso com o
  número. Recomendar desligar por padrão é resposta legítima.

#### Verificações além dos passos
- `git cat-file -t` em cada SHA citado, com a saída colada.
- `pwsh -File scripts/audit_reports.ps1` sem achados nas seções novas — achados
  antigos de outros marcos não contam e devem ser distinguidos.
- `pwsh -File scripts/check_doc_refs.ps1` limpo.
- UTF-8 validado em todo `.md` tocado.

#### Regras de execução
Nenhum número entra sem o comando que o produziu colado ao lado. Ledger antes de
reportar conclusão.

#### Contrato de relatório
Esta tarefa **não tem prova de mutação**: não envia código.

**Files:** `docs/ARCHITECTURE.md`, `docs/PRD.md`, `README.md`, ledger
**Commit:** `docs: record the multi-instance measurements and the daemon decision`

---

## Ordem da Parte II

```
88 -> 89 -> 90 -> 91 -> 92 -> 93
```

- **88 primeiro**: independente, e o maior ganho por linha mexida.
- **89 independente de 90 a 92**, mas depois de 88 para as medições de RSS não
  se confundirem.
- **90 é bloqueante para 91 e 92.** Abrir socket antes de reescrever a garantia
  faz o gate reprovar, e gate desligado é pior que gate ausente.
- **92 por último entre as de código**: maior risco, e quer a 91 estável.

## Adendo ao prompt de despacho

> **A Parte II reabre uma decisão fechada.** O RNF-30 dizia "nenhum socket de
> saída em nenhuma circunstância", com razão de produto escrita: o cofre pode
> conter material confidencial e a garantia precisa ser verificável. O dono do
> projeto autorizou reabrir em 2026-08-05. **A Task 90 reescreve a garantia de
> modo a continuar auditável em um comando, e é bloqueante para 91 e 92.** Quem
> executar 91 ou 92 antes da 90 vai encontrar o `check_net` vermelho e a
> tentação de desligá-lo; desligar o gate é o pior desfecho possível desta
> batelada.
>
> **A escolha do transporte já foi medida e não se re-litiga:** AF_UNIX, 25,7 µs
> contra 82,9 µs do named pipe em 256 B, na biblioteca padrão, mesmo código nos
> três sistemas. Ver D-M7-6.
>
> **A Task 92 pode terminar em "não compensa".** Se o RSS agregado de três
> sessões não cair o suficiente para justificar um processo de longa vida a mais,
> o relatório diz isso com o número e a Task 93 recomenda desligar por padrão.
> Resultado medido que contraria a expectativa é resultado, não falha.
>
> **Aceitação por tarefa da Parte II:**
> - **88** — falha barata: `Once` que trava o erro para sempre. Exigir o segundo
>   caso do teste, o da falha transitória.
> - **89** — falha barata: medir RSS de uma instância e chamar de ganho. Só o
>   agregado de três prova compartilhamento.
> - **90** — falha barata: analisador que aceita rede vinda de variável. Exigir
>   as seis saídas de disparo.
> - **91** — falha barata: sem fallback em processo, um socket quebrado inutiliza
>   a ferramenta. Exigir os dois testes de queda.
> - **92** — falha barata: dez pontes iniciando dez daemons. Exigir o teste.
> - **93** — sem prova de mutação, e o relatório tem de dizer isso.
