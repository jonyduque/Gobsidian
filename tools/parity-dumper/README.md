# Parity Dumper

Este plugin não faz parte da distribuição oficial. Ele existe apenas para extrair o estado real do cache de metadados do Obsidian em um cofre de teste, para que nossa bateria de testes de paridade (em `internal/index/parity_test.go`) não baseie suas expectativas no que *achamos* que o Obsidian faz, mas no que ele de fato faz.

## Como usar

### 1. Compilar

```bash
cd tools/parity-dumper
npx esbuild main.ts --bundle --outfile=main.js --format=cjs --platform=browser --external:obsidian
```

**`--external:obsidian` não é opcional.** O módulo `obsidian` não existe no npm: o próprio aplicativo o injeta em tempo de execução. Sem a flag, o esbuild tenta resolvê-lo e falha com `Could not resolve "obsidian"`.

**`--platform=browser`, não `node`.** Um plugin do Obsidian roda no renderer do Electron. Com `platform=node` o esbuild resolve builtins de Node que não existem ali.

O esbuild não faz checagem de tipos, então não é preciso instalar `obsidian` como dependência de desenvolvimento só para compilar.

### 2. Instalar e rodar

1. Copie `main.js` e `manifest.json` — **não** o `main.ts` — para `<cofre-de-teste>/.obsidian/plugins/parity-dumper/`
2. Em Configurações → Plugins da comunidade, desative o Modo Restrito e habilite o **Parity Dumper**
3. Espere a indexação do cofre terminar. O `metadataCache` é preenchido de forma assíncrona, e rodar o comando cedo demais produz uma referência parcial — que é pior que nenhuma, porque parece completa
4. Paleta de comandos → **Dump metadata cache to JSON**
5. Mova o `metadata.json` gerado na raiz do cofre para `testdata/parity/`, e as notas do cofre para `testdata/parity/vault/`

### 3. Conferir antes de confiar

O teste de paridade pula quando o corpus ou a referência estão vazios, justamente para não afirmar uma paridade que ninguém verificou. Confira que ele passou a **rodar**:

```bash
go test ./internal/index/ -run TestParity -v
```

`SKIP` significa que o corpus ou o `metadata.json` continuam vazios. `PASS` sem `SKIP` significa que a comparação aconteceu de verdade.

## Casos Divergentes Documentados
*(Se encontrarmos casos onde o comportamento do Obsidian seja considerado incorreto por nós e não formos simular 1:1, documente aqui antes de ignorar no teste.)*
