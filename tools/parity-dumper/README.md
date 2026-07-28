# Parity Dumper

Este plugin não faz parte da distribuição oficial. Ele existe apenas para extrair o estado real do cache de metadados do Obsidian em um cofre de teste, para que nossa bateria de testes de paridade (em `internal/index/parity_test.go`) não baseie suas expectativas no que *achamos* que o Obsidian faz, mas no que ele de fato faz.

## Como usar

1. Copie esta pasta para `<cofre-de-teste>/.obsidian/plugins/parity-dumper/`
2. Compile com `esbuild`: `npx esbuild main.ts --bundle --outfile=main.js --format=cjs --platform=node`
3. Habilite o plugin no Obsidian.
4. Rode o comando "Dump metadata cache to JSON".
5. Mova o arquivo `metadata.json` gerado na raiz do cofre para `testdata/parity/`.

## Casos Divergentes Documentados
*(Se encontrarmos casos onde o comportamento do Obsidian seja considerado incorreto por nós e não formos simular 1:1, documente aqui antes de ignorar no teste.)*
