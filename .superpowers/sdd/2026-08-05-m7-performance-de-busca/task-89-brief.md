# Task 89 — Arena de posições mapeada do arquivo

**Tier: modelo forte.** Envolve `unsafe` e comportamento de mapeamento de
memória por sistema; o modo de falha é corrupção silenciosa, não lentidão.

#### Onde encaixa
Depois da 88. É a única mudança que faz N instâncias custarem **menos que N
vezes** o custo de uma.

#### O que vincula esta tarefa

Repetido aqui de propósito: o brief é a unidade que viaja, e decisão citada por
código fica no preâmbulo, que não viaja com ela.

- **Otimização que muda resultado é defeito, não trade-off.** O golden de
  ranking da Task 78 tem de ficar idêntico. **Nunca regenerar com `-update`
  para fazer passar.**
- **`benchstat` com `-count=6`, uma mudança por vez.** `~` reverte a mudança.
- **Nenhum teto de RNF é afrouxado nesta batelada.**

#### A evidência
O array de posições é **~291 MB** dos ~382 MB em repouso no cofre de referência
(18.229.295 posições vezes 16 bytes). Hoje cada processo aloca a sua cópia.
Mapeado do arquivo em modo leitura, o cache de páginas do sistema operacional o
compartilha entre processos: dez instâncias pagariam cerca de uma vez a arena.

#### A decisão que esta tarefa tem de acertar
**O formato já favorece isto e não deve ser refeito.** As posições no formato 5
estão em varint com delta — comprimidas, e portanto **não mapeáveis direto**.
Duas saídas, e a escolha é desta tarefa:

- **(a)** gravar uma segunda seção com as posições em formato fixo de 16 bytes,
  alinhada, para ser mapeada; o varint continua para quem não puder mapear.
  Custa espaço em disco, cerca de 291 MB.
- **(b)** manter só o varint e mapear o arquivo comprimido, decodificando sob
  demanda por posting. Economiza disco, mas cada leitura decodifica.

**Pré-decidido: comece por (a) e meça.** A troca é disco por memória
compartilhada, e disco é o recurso barato aqui. Se (a) não reduzir o RSS
agregado de três instâncias em pelo menos 30%, **pare e reporte** — não parta
para (b) sem uma decisão nova.

#### Armadilhas já pagas que se aplicam
- **O cofre fica em OneDrive.** Arquivo mapeado que o sincronizador mexe embaixo
  é classe de falha que este projeto ainda não pagou. O **cache** fica fora do
  cofre, em `%LOCALAPPDATA%`, o que evita isso — **confirmar** que continua fora,
  e recusar mapear se o caminho do cache estiver dentro do cofre.
- **Capacidade travada nas subfatias** já é regra do projeto. Num array mapeado
  em modo leitura, um append que escrevesse por cima dispararia falha de
  proteção de página, que é um crash e não um dado errado. Manter mesmo assim.
- **`unsafe` sem prova de benchmark é injustificado.** Aqui a prova exigida não
  é velocidade: é RSS agregado de várias instâncias.

#### Verificações além dos passos
- **RSS agregado de três instâncias simultâneas** no mesmo cofre, antes e
  depois. É a medida que a tarefa existe para mover; RSS de uma instância só não
  prova compartilhamento nenhum.
- Cache invalidado com o arquivo mapeado aberto: o `os.Rename` do salvamento
  atômico **falha no Windows** se alguém tem o arquivo mapeado. Testar, e decidir
  o que acontece — desmapear antes de regravar é o caminho provável.
- Arquivo truncado ou corrompido: recusa, não mapeia lixo.

#### Regras de execução
Rodar `pwsh -File scripts/verify.ps1` antes de dizer que acabou. Registrar no
ledger antes de reportar conclusão. Escopo não encolhe em silêncio.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/search/mmap.go `
  -Anchor 'if dentroDoCofre(caminhoCache, vaultPath) {' -Replacement 'if false {' `
  -Test TestRecusaMapearCacheDentroDoCofre -Package ./internal/search/
```

#### Contrato de relatório
RSS de uma e de **três** instâncias, antes e depois. Resultado do teste de
regravação com o arquivo mapeado. Se o ganho agregado ficar abaixo de 30%,
**dizer o número e parar** — é resposta melhor que seguir para (b) sozinho.

**Files:** `internal/search/mmap.go`, `internal/search/mmap_windows.go`,
`internal/search/mmap_unix.go`, `internal/search/persist_codec.go`, testes
**Commit:** `perf(search): map the position array from the cache file`

---

