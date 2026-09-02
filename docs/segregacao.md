# segregacao.md — como o código se agrupa por função e por proximidade

Levantamento de 2026-09-02. **Nada aqui foi implementado** — é leitura do que
existe, mais três candidatos a extração com o custo de cada um. Decisão do dono
pendente.

Os números vieram de medição, não de leitura por cima:

- arestas: `go list -f '{{.ImportPath}} -> {{join .Imports " "}}' ./...`,
  que não enxerga arquivo de teste;
- loc: contagem de linha por pacote, arquivo de teste fora;
- duplicação: comparação dos corpos das funções, não dos nomes.

---

## O grafo de produção é mais apertado que o documentado

Esta foi a primeira coisa que a medição derrubou. Até 2026-09-02 o grafo do
`CLAUDE.md` somava arestas de produção com arestas de teste numa lista só:

```
daemon → config, ipc, mcpsrv          (o doc dizia também service, vault)
mcpsrv → config, index, parser, service, vault   (o doc dizia também search, watcher)
```

`daemon` não conhece `service` nem `vault`: ele fala com `mcpsrv` e mais nada do
domínio. `mcpsrv` não conhece `search` nem `watcher`. As quatro arestas a mais
existem só em arquivo de teste, e teste pode montar o mundo inteiro sem que isso vire
acoplamento do produto.

Importa porque é esse grafo que autoriza aresta nova. Um grafo que já concede o
import faz a revisão aprovar sem perguntar. `CLAUDE.md` foi corrigido, e agora
separa as duas listas.

---

## Por função — sete camadas, 20.565 loc

| Grupo | Pacotes | loc | Pergunta que responde |
|---|---|---|---|
| **Chão** | `vault`, `text` | 918 | Onde fica, como se escreve, é seguro abrir? |
| **Compreensão** | `parser`, `index`, `search` | 8.514 | O que os bytes dizem? (só lê) |
| **Mutação** | `writer` | 985 | Como troco bytes sem perder nota |
| **Frescor** | `watcher` | 875 | O disco mudou; o índice acompanha |
| **Casos de uso** | `service` | 3.097 | As operações, em tipos de domínio |
| **Protocolo** | `mcpsrv`, `ipc` | 1.516 | Traduz fio ↔ domínio |
| **Processo** | `cmd`, `daemon`, `doctor`, `lifecycle`, `config`, `console` | 4.660 | Nada sobre Obsidian: travas, sinais, flags, terminal |

O corte que carrega peso é **Compreensão × Mutação**: 8.514 loc que só leem
contra 985 que escrevem. Nenhum pacote do primeiro grupo abre arquivo para
escrita. É a cintura mais defensável do projeto, e já está de pé — não há o que
fazer além de não estragar.

Fan-in de produção: `vault` 7, `parser` 5, `index` 4. O eixo do desenho é
`vault.CanonicalPath`, o tipo que atravessa todas as camadas, e não `service`.

---

## Por proximidade — onde a árvore discorda do agrupamento

Três funções moram em mais de um lugar. Só a primeira paga a mudança.

### 1. Persistência binária — 2.219 loc, dois motores para o mesmo problema

`index/persist_codec.go` (817 loc) e `search/persist_codec.go` (653 loc) têm
cada um seu par `escritor`/`leitor`, com **as mesmas primitivas de mesmo nome**:
`uvarint`, `varint`, `str`, `falha`. Os corpos coincidem — mesmo
`binary.PutUvarint` para o mesmo buffer; um escreve direto no `io.Writer`, o
outro passa por um método intermediário. São dois tipos `CacheHeader` que
diferem por um campo: `AnalyzerVersion` de um lado, `AssetCount` do outro.

Com `persist.go` e o `mmap` dos dois lados, são 2.219 loc — **34% de
`index`+`search`**, 11% do repositório.

O custo de manter assim não é o tamanho, é a divergência: dois formatos
versionados evoluindo em separado, e todo defeito de codec precisa ser achado
duas vezes. É o único caso deste documento em que a duplicação é de
**mecanismo**, não de coincidência.

Extração natural: um pacote com as primitivas — varint, string, checksum,
cabeçalho versionado, arena mmap —, com `index` e `search` guardando apenas o
próprio esquema. Cria uma folha nova, que é o tipo de aresta que o `CLAUDE.md`
manda justificar; a justificativa é esta seção.

### 2. Troca atômica — três implementações

`writer/atomic.go:141`, `index/persist.go:124`, `search/persist.go:87`.

A do `writer` é a cuidadosa: restaura o modo do alvo, porque `os.CreateTemp`
cria 0600 e um rename por cima trocaria a permissão da nota do usuário; e trata
a repetição no Windows. As duas de cache são a versão curta.

A regra do projeto — *todo caminho derivado passa por UMA função* — aponta para
cá. Mas o consumo difere no que importa: cache perdido se refaz, nota do usuário
não. Unificar as três acopla o caminho barato ao caro sem que o caro fique mais
seguro. Recomendação: só junto com o item 1, e só entre os dois caches.

### 3. Chave derivada — a tag escapa da conta única

`index/chave.go` é explicitamente a conta única e cumpre o que promete para
caminho, nome de arquivo e alias. Mas `index/query.go` tem 10 `strings.ToLower`
soltos: parte é parâmetro de consulta (`Sort`, `Order`, `TagMode`), o que está
certo, e parte é comparação de tag e de chave de campo inline — que baixa caixa
sem normalizar Unicode.

Não é o defeito que `chave.go` fechou; é o mesmo mecanismo um nível acima. Um
cofre com `#Ação` gravado em NFD e consultado em NFC tem, aqui, a mesma classe
de falha silenciosa. **Não medido** em cofre real — antes de mexer, vale contar
quantas tags do corpus não são ASCII.

---

## O que não separar

- **`text`** — 62 loc, um arquivo, nenhum teste. Pequeno demais para virar
  debate, e fan-in 3 justifica existir como está.
- **Arquivos por build tag** — `mmap_windows.go`/`mmap_unix.go` em `search`,
  `trava_windows.go`/`trava_unix.go` em `daemon`. Já estão certos: perto do uso, e não
  reunidos num pacote "plataforma" que obrigaria a olhar em dois lugares.
- **`service`, 3.097 loc** — parece grande, divide bem por arquivo
  (`write.go` 841, `graph.go` 747, `search.go` 540). Fachada gorda não é pacote
  confuso.

---

## Se virar tarefa

Só o item 1 tem ganho medível: uma superfície de codec em vez de duas, um
formato a versionar em vez de dois. Os itens 2 e 3 são registro — o 2 porque a
recomendação é **não** unificar tudo, e essa decisão precisa estar escrita para
não ser retomada do zero; o 3 porque falta a medição que diria se vale.
