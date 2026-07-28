var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

// main.ts
var main_exports = {};
__export(main_exports, {
  default: () => ParityDumper
});
module.exports = __toCommonJS(main_exports);
var import_obsidian = require("obsidian");
var SCHEMA_VERSION = 2;
var ParityDumper = class extends import_obsidian.Plugin {
  async onload() {
    this.addCommand({
      id: "dump-metadata-cache",
      name: "Dump metadata cache to JSON",
      callback: async () => {
        const notes = {};
        for (const file of this.app.vault.getMarkdownFiles()) {
          const fc = this.app.metadataCache.getFileCache(file);
          if (!fc) continue;
          notes[file.path] = {
            headings: (fc.headings ?? []).map((h) => ({
              level: h.level,
              heading: h.heading
            })),
            tags: (fc.tags ?? []).map((t) => t.tag.replace(/^#/, "")),
            frontmatterTags: fc.frontmatter?.tags ?? null,
            aliases: fc.frontmatter?.aliases ?? null,
            blocks: Object.keys(fc.blocks ?? {}),
            links: (fc.links ?? []).map((l) => ({
              link: l.link,
              displayText: l.displayText
            })),
            embeds: (fc.embeds ?? []).map((e) => ({ link: e.link }))
          };
        }
        const mc = this.app.metadataCache;
        const dump = {
          schema: SCHEMA_VERSION,
          generatedAt: (/* @__PURE__ */ new Date()).toISOString(),
          notes,
          resolvedLinks: mc.resolvedLinks ?? {},
          unresolvedLinks: mc.unresolvedLinks ?? {}
        };
        await this.app.vault.adapter.write(
          "metadata.json",
          JSON.stringify(dump, null, 2)
        );
        const arestas = Object.values(dump.resolvedLinks).reduce(
          (n, alvos) => n + Object.keys(alvos).length,
          0
        );
        new import_obsidian.Notice(
          `metadata.json: ${Object.keys(notes).length} notas, ${arestas} arestas resolvidas`
        );
      }
    });
  }
};
