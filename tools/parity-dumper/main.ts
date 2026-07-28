import { Plugin, TFile, Notice } from "obsidian";

// Plugin de desenvolvimento descartavel. Nao e parte do produto, nao e
// distribuido. Existe para que a metrica de paridade seja o comportamento
// real do Obsidian, e nao a nossa interpretacao da documentacao dele.
export default class ParityDumper extends Plugin {
  async onload() {
    this.addCommand({
      id: "dump-metadata-cache",
      name: "Dump metadata cache to JSON",
      callback: async () => {
        const out: Record<string, unknown> = {};

        for (const file of this.app.vault.getMarkdownFiles()) {
          const cache = this.app.metadataCache.getFileCache(file);
          if (!cache) continue;

          out[file.path] = {
            headings: (cache.headings ?? []).map((h) => ({
              level: h.level,
              heading: h.heading,
            })),
            tags: (cache.tags ?? []).map((t) => t.tag.replace(/^#/, "")),
            frontmatterTags: cache.frontmatter?.tags ?? null,
            aliases: cache.frontmatter?.aliases ?? null,
            blocks: Object.keys(cache.blocks ?? {}),
            links: (cache.links ?? []).map((l) => ({
              link: l.link,
              displayText: l.displayText,
              resolved: this.resolve(l.link, file),
            })),
            embeds: (cache.embeds ?? []).map((e) => ({
              link: e.link,
              resolved: this.resolve(e.link, file),
            })),
          };
        }

        await this.app.vault.adapter.write(
          "metadata.json",
          JSON.stringify(out, null, 2),
        );
        new Notice("metadata.json gravado na raiz do cofre");
      },
    });
  }

  private resolve(link: string, from: TFile): string | null {
    const target = this.app.metadataCache.getFirstLinkpathDest(
      link.split("#")[0],
      from.path,
    );
    return target ? target.path : null;
  }
}
