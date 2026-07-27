package vault

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// Entry e uma nota ou anexo encontrado na varredura. Path ja vem com a
// grafia real do disco — esta e a camada que a produz, porque CanonicalPath
// sozinho preserva a grafia que o chamador passou, nao a que existe.
//
// CloudOnly marca o arquivo que o sincronizador ainda nao baixou. E lido do
// atributo, nunca abrindo o arquivo: abrir dispara a hidratacao que o campo
// existe para evitar.
type Entry struct {
	Path      CanonicalPath
	Size      int64
	ModTime   time.Time
	IsNote    bool
	CloudOnly bool
}

// excludedDirs sao podados na varredura, nunca descendidos.
var excludedDirs = map[string]bool{
	".obsidian": true,
	".git":      true,
	".trash":    true,
	".stfolder": true,
}

// assetExts sao indexados por nome, nunca lidos (PRD RF-60). Sem eles, todo
// embed de imagem vira link quebrado e vault_stats fica dominado por falsos
// positivos.
var assetExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".svg": true, ".bmp": true,
	".pdf": true, ".canvas": true,
	".mp3": true, ".wav": true, ".m4a": true, ".ogg": true,
	".mp4": true, ".webm": true, ".mov": true,
}

// Arquivos de controle de sincronizador e de editor, que geram evento e nunca
// interessam (WINDOWS.md §1.4).
func isNoise(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(name, "~$"):
		return true
	case strings.HasPrefix(lower, ".~lock."):
		return true
	// desktop.ini, Thumbs.db e .DS_Store sao nomes fixos cuja extensao nunca
	// e ".md" nem uma extensao de assetExts (".ini", ".db" ou nenhuma
	// extensao, respectivamente). O filtro de extensao logo abaixo ja os
	// descartaria de qualquer forma, com ou sem este ramo — defensivo
	// (belt-and-braces), nao alcancavel por efeito. Nao existe fixture capaz
	// de provar este ramo sem inventar um nome que o Windows nunca produz.
	case lower == "desktop.ini", lower == "thumbs.db", lower == ".ds_store":
		return true
	// *.tmp: ".tmp" nao esta em assetExts, entao qualquer arquivo que caia
	// aqui ja seria descartado pelo filtro de extensao mesmo sem isNoise.
	// Defensivo pelo mesmo motivo do ramo acima, nao alcancavel por efeito.
	case strings.HasSuffix(lower, ".tmp"):
		return true
	case strings.HasPrefix(name, ".gobsidian-tmp-"):
		return true
	}
	return false
}

// Walk percorre o cofre aplicando as exclusoes e classificando cada arquivo
// como nota ou anexo. Arquivos que nao sao nem um nem outro sao ignorados.
func (v *Vault) Walk(ctx context.Context, fn func(Entry) error) error {
	return filepath.WalkDir(v.walkRoot, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			// d == nil significa que a falha foi na propria raiz: WalkDir nao
			// conseguiu nem fazer Lstat nela. Isso acontece quando o cofre
			// some entre New e Walk — unidade removivel desconectada, pasta
			// sincronizada que o cliente de nuvem moveu, share de rede caido.
			//
			// Engolir esse erro faria Walk devolver sucesso com zero entradas,
			// e o servidor reportaria com confianca que o cofre esta vazio.
			// Um cofre inacessivel e um erro; um cofre vazio e um fato. As
			// duas coisas nao podem produzir a mesma resposta.
			if d == nil {
				return fmt.Errorf("varrendo a raiz do cofre %q: %w", v.root, err)
			}
			// Um diretorio ilegivel nao derruba a varredura inteira. O cofre
			// e do usuario, e uma pasta com ACL estranha e problema local.
			if d.IsDir() {
				v.recordSkip(abs, err)
				return fs.SkipDir
			}
			v.recordSkip(abs, err)
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		name := d.Name()

		if d.IsDir() {
			if abs == v.walkRoot {
				return nil
			}
			if excludedDirs[strings.ToLower(name)] {
				return fs.SkipDir
			}
			return nil
		}

		if isNoise(name) {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(name))
		isNote := ext == ".md"
		isAsset := assetExts[ext]
		if !isNote && !isAsset {
			return nil
		}

		canon, cErr := Canonicalize(v.walkRoot, abs)
		if cErr != nil {
			v.recordSkip(abs, cErr)
			return nil
		}
		info, iErr := d.Info()
		if iErr != nil {
			v.recordSkip(abs, iErr)
			return nil
		}

		return fn(Entry{
			Path:      canon,
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			IsNote:    isNote,
			CloudOnly: IsCloudOnly(abs),
		})
	})
}
