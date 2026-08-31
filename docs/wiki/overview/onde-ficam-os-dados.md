---
title: Onde ficam os dados
type: overview
status: stale
description: Caches, socket, lock, lixeira e temporários — o que o servidor grava e onde.
source_paths:
  - internal/config/config.go
  - internal/index/persist.go
  - internal/search/persist.go
  - internal/ipc/ipc.go
  - internal/writer/atomic.go
source_commit: b2be492
tags: [cache, dados, disco]
language: pt-BR
updated_at: '2026-08-16'
---

# Onde ficam os dados

Regra geral: **nada do servidor é gravado dentro do cofre**, exceto as notas que
o usuário mandou escrever e a lixeira. O cache fora do cofre é decisão fechada
(PRD D1) e tem um motivo prático — o cofre de referência fica no OneDrive, e um
arquivo que o sincronizador mexe embaixo é uma classe de falha inteira.

## A chave do cofre

`config.VaultKey(vaultPath)` é o **xxhash do caminho absoluto em minúsculas**, em
hexadecimal. É a única função que calcula essa chave, e três coisas dependem
dela: o diretório de cache, o caminho do socket e o do lock. Duas contas
separadas do mesmo hash concordariam por coincidência até uma delas mudar
sozinha — é o padrão que já custou caro aqui.

## Cache

```
<os.UserCacheDir()>/gobsidian/<VaultKey>/
├── index_cache.gob       índice de metadados
└── inverted_cache.gob    índice de busca
```

Se `os.UserCacheDir()` falhar, cai para `os.TempDir()`.

Os dois são gravados **atomicamente**: temporário no mesmo diretório, depois
`os.Rename`. Os dois carregam versão de formato **e** de parser: corrigir um bug
de parsing e reiniciar carregaria de volta o índice errado, porque mtime e
tamanho dos arquivos não mudaram.

A extensão `.gob` do cache de busca é **legado de caminho**: o conteúdo é um
codec binário próprio desde o formato 5. Ver
[Formato do cache](../entities/formato-do-cache.md).

Os dois validam **cobertura**, não só versão. Um cabeçalho que promete mais do
que o corpo trouxe é recusado — a construção grava parciais de propósito, e
aceitar um parcial como completo faria a busca servir menos notas do que existem.

## Socket e lock do daemon

```
<diretório de runtime do usuário>/
├── <VaultKey>.sock       socket AF_UNIX
└── <VaultKey>.sock.lock  lock de inicialização, com o PID dentro
```

Em Unix o socket é restrito a `0600`; no Windows a garantia vem da ACL herdada do
diretório. O lock guarda o PID justamente para poder ser **reivindicado** quando
o dono morreu — um lock com PID morto já deixou o daemon desligado por três dias.

Ver [Daemon e ponte](../features/daemon-e-ponte.md).

## Dentro do cofre

| O quê | Onde | Quem cria |
|---|---|---|
| Lixeira | `.trash/` | `note_delete` com `to_trash` |
| Temporários de escrita | `.gobsidian-tmp-*` no mesmo diretório do alvo | `writer.WriteAtomic` |

`.trash` é um dos diretórios **excluídos da varredura** (junto de `.obsidian`,
`.git` e `.stfolder`), então nota na lixeira não volta ao índice. Isso também
significa que hoje **não há tool para listar ou restaurar** o que foi para lá.

Temporário órfão só sobra quando o processo é morto sem rodar `defer`.
`writer.SweepStaleTempFiles` limpa isso **no boot**, que é o único momento sem
escrita em voo — varrer durante uma escrita apagaria o temporário de outra.

## Ver também

- [Escrita](../features/escrita.md)
- [Armadilhas já pagas](../risks/armadilhas-pagas.md)
