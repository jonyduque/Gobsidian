# SDD ledger — plan: docs/superpowers/plans/2026-09-27-cli-das-tools.md

Spec: docs/superpowers/specs/2026-09-25-cli-das-tools-design.md (revisto em 2026-09-27). Pedido do dono em 2026-09-27: "Segue como sugerido, faça tudo na ordem sugerida até encerrar." Decisão do dono na mesma data: `vaults` lista os cofres, `config` configura os hosts; o grupo `vault` sai e stats/search/broken-links vão para a raiz (opção 1 proposta e aceita).

BASE inicial: d877840. Branch: master (o dono trabalha em master). Executado pelo orquestrador, sem subagente.

## Tasks
- Desenho revisto e plano escritos (2026-09-27).
- Parte K (K1-K7), 2026-09-27: `config` = o `vaults` antigo (textos.ResumoConfig com o texto que o dono escreveu para ele); `vaults` novo em cmd/gobsidian/vaults.go: listarCofres une instalar.CofresDoObsidian e configuracaoAtualFn por instalar.CaminhoCanonicoDeCofre; terminal = moldura (nome + estado numa linha, caminho apagado na de baixo), fora do terminal = lista JSON numa linha. saida.go novo e compartilhado com a Parte L: opcoesDeSaida (--json/--texto, destino por console.EhTerminal -- terminal, nao cor), erroComCodigo e codigos 0/1/2; main sai com o codigo; SetFlagErrorFunc da raiz marca flag errada como uso (2); carregarConfig marca cofre que nao resolve como uso (2); `path` sem --add/--remove tambem.
- Defeito achado no teste da K: na primeira redacao o estado vinha depois do caminho, e a moldura (teto de 78 colunas) cortava "configurado" nos caminhos longos -- TestVaultsComTextoMostraATabela reprovou. O caminho foi para a linha de baixo.
- Mutacoes K (mutate.ps1, restauracao por SHA-256): MK1 uniao pela grafia crua -> TestListarCofresUnePelaGrafiaCanonica FAIL ("quer 3 cofres (Estudo uma vez so)", saiu 4); MK2 configurado fora do Obsidian some -> mesmo teste FAIL (saiu 2); MK3 fora do terminal sai texto -> TestVaultsForaDoTerminalSaiEmJSON FAIL; MK4 --json e --texto juntos aceitos -> TestJSONETextoJuntosSaoUsoErrado FAIL ("codigo = 1; quer erro de uso (2)").
- Docs K: README (en e pt-BR: config no lugar de vaults, e a linha de vaults), TEXTOS.md (secao config, secao vaults nova, FlagJSON/FlagTexto), ESTRUTURA.md (vaults.go, saida.go).
- verify.ps1 da K: EXIT=0, "Bateria completa. Pode commitar." (7 testes pulados, informativo).
