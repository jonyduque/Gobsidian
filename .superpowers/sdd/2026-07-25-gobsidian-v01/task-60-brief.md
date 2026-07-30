### Task 60: `--read-only` verificado no `ListTools`

RF-51.

#### A decisão que já está fechada

**As tools de escrita ficam AUSENTES do `ListTools`, não presentes-e-rejeitadas.** Um modelo que vê a tool na lista vai chamá-la; receber erro faz ele tentar variações da chamada, o que gasta contexto e não chega a lugar nenhum. Ausência é a informação correta.

#### A armadilha

**É a irmã do campo de valor fixo.** Um teste que afirma "chamar a tool em read-only devolve erro" passa com a tool listada — e o defeito é justamente ela estar listada. **O teste tem de afirmar a lista**, não o comportamento da chamada.

E: `--read-only` também não pode desligar o watcher. Modo leitura com índice desatualizado é pior, não melhor. Já está escrito na Task 29; confirme que continua verdade.

#### Verificações além dos passos

- Com `--read-only`, `ListTools` **não** contém `note_create`, `note_append`, `note_patch`. Afirme a lista, nome por nome.
- Sem `--read-only`, contém as três.
- As tools de **leitura** continuam presentes nos dois modos.
- O watcher continua ativo com `--read-only`? Afirme por `vault_stats`.
- `ReadOnlySet` está preenchido nos dois subcomandos (`serve` e `doctor`)? `grep -rn "ReadOnlySet"` — esquecer em um faz a flag virar no-op silencioso.

**Prova de mutação obrigatória:** faça o registro ignorar `--read-only` e confirme que o teste de `ListTools` reprova.

**Files:** Modify `internal/mcpsrv/`, testes
**Commit:** `feat(mcpsrv): write tools absent from ListTools under read-only`

---

