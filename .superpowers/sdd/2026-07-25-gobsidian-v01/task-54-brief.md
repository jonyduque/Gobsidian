### Task 54: `internal/writer/lock.go` — trava por caminho

#### Onde isto encaixa

Primeira peça de `internal/writer`, que não existe. Sem ela, duas chamadas MCP concorrentes sobre a mesma nota fazem *lost update*: a segunda lê antes de a primeira gravar, e a escrita da primeira desaparece sem erro.

#### O que implementar

Um registro de mutexes por `vault.CanonicalPath`. Trava por **caminho**, não global: escrever em `Civil/a.md` não pode bloquear `Penal/b.md`, senão o servidor serializa tudo.

**A armadilha é o vazamento.** Um `map[CanonicalPath]*sync.Mutex` que nunca remove entrada cresce com o número de caminhos já escritos e não volta. Um que remove logo depois do `Unlock` tem corrida: outra goroutine pode ter pegado o ponteiro e estar esperando nele. **Use contagem de referências**, ou aceite o crescimento e diga por quê — as duas são defensáveis, e escolher em silêncio não é.

**A chave é `CanonicalPath`, e casing importa.** Em Windows, `Civil/A.md` e `civil/a.md` são o mesmo arquivo e duas chaves diferentes. Duas travas para o mesmo arquivo é o mesmo que nenhuma. `index.lowerPath` existe para isso; use a mesma normalização, através de **uma** função — chave derivada calculada em dois lugares diverge, e foi o que aconteceu com `byAlias`.

#### Verificações além dos passos

- Duas escritas concorrentes na **mesma** nota: a segunda espera, e as duas mudanças sobrevivem? `go test -race`.
- Duas escritas concorrentes em notas **diferentes**: rodam em paralelo? Meça, não suponha — trava global passa no primeiro teste e reprova aqui.
- `Civil/A.md` e `civil/a.md` pegam a **mesma** trava?
- Depois de N escritas em N caminhos distintos, quantas entradas sobram no registro? Um número medido.

**Prova de mutação obrigatória:** troque a trava por caminho por uma trava global e confirme que o teste de paralelismo reprova. Depois remova a trava inteira e confirme que o teste de *lost update* reprova. **Duas mutações, duas saídas coladas.**

**Files:** Create `internal/writer/lock.go`, `lock_test.go`
**Commit:** `feat(writer): per-path write lock`

---

