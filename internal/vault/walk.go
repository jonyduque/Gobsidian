package vault

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

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
	case lower == "desktop.ini", lower == "thumbs.db", lower == ".ds_store":
		return true
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
	return filepath.WalkDir(v.root, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			// Um diretorio ilegivel nao derruba a varredura inteira. O cofre
			// e do usuario, e uma pasta com ACL estranha e problema local.
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		name := d.Name()

		if d.IsDir() {
			if abs == v.root {
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

		canon, cErr := Canonicalize(v.root, abs)
		if cErr != nil {
			return nil
		}
		info, iErr := d.Info()
		if iErr != nil {
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
