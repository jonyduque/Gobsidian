# Task 91 — Transporte IPC e o processo-ponte

**Tier: modelo forte.** O modo de falha barato é a ponte que não sabe cair para
o modo em processo — e aí um socket quebrado inutiliza a ferramenta.

#### Onde encaixa
Depois da 90, que é bloqueante. Antes da 92.

#### O que vincula esta tarefa
**D-M7-6, decidida por medição em 2026-08-05:** AF_UNIX nos três sistemas, sem
compilação condicional para o transporte.

| Transporte | 256 B | 4 KB | 64 KB |
|---|---|---|---|
| AF_UNIX | 25,7 µs | 23,0 µs | 42,9 µs |
| named pipe | 82,9 µs | 93,5 µs | 110,0 µs |

AF_UNIX ganhou em todos os tamanhos, está na stdlib e é o mesmo código nos três
sistemas. E a escolha quase não importa: 25 µs contra uma busca de 90 a 200 ms.
**Não re-litigar sem medição nova.** Windows 10 1803 ou superior é requisito.

Vale também: **stdout pertence ao JSON-RPC**, todo log vai para stderr; e
**código de plataforma fica atrás de build tag, em arquivo separado**.

#### A decisão que esta tarefa tem de acertar
1. **A ponte é burra.** Ela copia bytes entre o stdin e o stdout que o host lhe
   deu e o socket. Não interpreta JSON-RPC, não tem índice, não tem estado. É o
   que a mantém em poucos MB.
2. **Fallback em processo é obrigatório.** Se o socket não existir, não conectar,
   ou a versão não bater, a ponte **serve ela mesma**, exatamente como hoje. Sem
   isso, um socket quebrado transforma a ferramenta em nada, e o usuário não tem
   como diagnosticar.
3. **Compilação condicional só para o caminho do socket e a limpeza dele.**
   Windows deixa arquivo que precisa ser removido; Unix idem, em diretório de
   runtime do usuário. `ipc_windows.go` e `ipc_unix.go`.
4. **Permissão do socket é a garantia que substitui a antiga.** `0600` em Unix.
   No Windows o arquivo herda a ACL do diretório, então o socket vai para um
   diretório do próprio usuário, e a tarefa **verifica** que outro usuário não
   consegue abrir. Um socket legível por qualquer um, para um daemon que lê o
   cofre, é pior que qualquer preocupação de rede.

#### Armadilhas já pagas que se aplicam
- **`io.TeeReader` não propaga EOF.** A ponte copia nos dois sentidos; usar
  `mirrorReader`, que faz `dst.CloseWithError(err)`.
- **Goroutine parada em `Read` não é desenrolável por cancelamento de context.**
  Vale para as duas direções da cópia.
- **`ctx.Canceled` no retorno do laço de serviço é encerramento normal.**

#### Verificações além dos passos
- Socket ausente: a ponte serve em processo, e o log **diz** que caiu para esse
  modo.
- Socket presente mas de versão diferente: mesma coisa.
- Outro usuário do sistema não abre o socket. Se não der para testar no
  ambiente, **dizer isso** em vez de afirmar que está seguro.
- Os três mecanismos de encerramento continuam valendo para a ponte:
  `pwsh -File scripts/test_orphans.ps1 -Cycles 100` verde nos três cenários.

#### Regras de execução
`verify.ps1` e o gate de órfãos antes de dizer que acabou. Ledger antes de
reportar. Escopo não encolhe em silêncio.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/ponte.go `
  -Anchor 'return serveEmProcesso(ctx, cfg, log)' `
  -Replacement 'return err' `
  -Test TestPonteCaiParaModoEmProcesso -Package ./cmd/gobsidian/
```

#### Contrato de relatório
RSS da ponte sozinha. Latência de uma chamada de tool através dela contra a
mesma chamada em processo. Saída dos três cenários de órfãos. Resultado do teste
de permissão, ou a frase de que não foi possível testar no ambiente.

**Files:** `cmd/gobsidian/ponte.go`, `internal/ipc/ipc.go`,
`internal/ipc/ipc_windows.go`, `internal/ipc/ipc_unix.go`, testes
**Commit:** `feat(ipc): bridge stdio to a local AF_UNIX socket, with in-process fallback`

---

