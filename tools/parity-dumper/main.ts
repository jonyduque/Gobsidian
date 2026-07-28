import { Plugin, Notice } from "obsidian";

// Plugin de desenvolvimento descartavel. Nao e parte do produto, nao e
// distribuido. Existe para que a metrica de paridade seja o comportamento real
// do Obsidian, e nao a nossa interpretacao da documentacao dele.
//
// SCHEMA 2 — o que mudou e por que.
//
// A versao 1 descobria o alvo de cada link com metadataCache.getFirstLinkpathDest.
// Essa API resolve por caminho e por nome de arquivo, e NAO consulta o campo
// aliases do frontmatter. O efeito apareceu na primeira rodada de paridade: uma
// nota declarando `aliases: [Terceiro]` e alcancada por [[Terceiro]] dentro do
// Obsidian, mas a referencia registrava resolved=null.
//
// Isso e pior que um dado faltando. Se alguem ligasse a comparacao de resolucao
// contra aquela referencia, cada alias viraria divergencia — e a reacao natural
// seria "corrigir" o nosso resolvedor para parar de resolver aliases, quebrando
// um requisito correto para casar com um instrumento defeituoso. A metrica se
// voltaria contra o produto que ela existe para verificar.
//
// resolvedLinks e unresolvedLinks sao o grafo que o proprio aplicativo usa,
// aliases incluidos. Sao a fonte certa.
const SCHEMA_VERSION = 2;

interface NoteMetadata {
  headings: { level: number; heading: string }[];
  tags: string[];
  frontmatterTags: unknown;
  aliases: unknown;
  blocks: string[];
  links: { link: string; displayText?: string }[];
  embeds: { link: string }[];
}

interface Dump {
  schema: number;
  generatedAt: string;
  notes: Record<string, NoteMetadata>;
  // origem -> alvo -> contagem. O grafo resolvido do proprio aplicativo.
  resolvedLinks: Record<string, Record<string, number>>;
  // origem -> alvo bruto -> contagem, para o que o Obsidian NAO resolveu.
  // Separar os dois e o que permite a paridade arbitrar o que e link quebrado
  // de verdade e o que e referencia externa — hoje nosso vault_stats conta URL
  // externa como quebrada, e essa referencia e quem decide se isso esta errado.
  unresolvedLinks: Record<string, Record<string, number>>;
}

export default class ParityDumper extends Plugin {
  async onload() {
    this.addCommand({
      id: "dump-metadata-cache",
      name: "Dump metadata cache to JSON",
      callback: async () => {
        const notes: Record<string, NoteMetadata> = {};

        for (const file of this.app.vault.getMarkdownFiles()) {
          const fc = this.app.metadataCache.getFileCache(file);
          if (!fc) continue;

          notes[file.path] = {
            headings: (fc.headings ?? []).map((h) => ({
              level: h.level,
              heading: h.heading,
            })),
            tags: (fc.tags ?? []).map((t) => t.tag.replace(/^#/, "")),
            frontmatterTags: fc.frontmatter?.tags ?? null,
            aliases: fc.frontmatter?.aliases ?? null,
            blocks: Object.keys(fc.blocks ?? {}),
            links: (fc.links ?? []).map((l) => ({
              link: l.link,
              displayText: l.displayText,
            })),
            embeds: (fc.embeds ?? []).map((e) => ({ link: e.link })),
          };
        }

        const mc = this.app.metadataCache as unknown as {
          resolvedLinks?: Record<string, Record<string, number>>;
          unresolvedLinks?: Record<string, Record<string, number>>;
        };

        const dump: Dump = {
          schema: SCHEMA_VERSION,
          generatedAt: new Date().toISOString(),
          notes,
          resolvedLinks: mc.resolvedLinks ?? {},
          unresolvedLinks: mc.unresolvedLinks ?? {},
        };

        await this.app.vault.adapter.write(
          "metadata.json",
          JSON.stringify(dump, null, 2),
        );

        const arestas = Object.values(dump.resolvedLinks).reduce(
          (n, alvos) => n + Object.keys(alvos).length,
          0,
        );
        new Notice(
          `metadata.json: ${Object.keys(notes).length} notas, ${arestas} arestas resolvidas`,
        );
      },
    });
  }
}
