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

// IsExcludedDir verifica se o nome de um diretorio esta na lista de excluidos (.git, .obsidian, etc).
func IsExcludedDir(name string) bool {
	return excludedDirs[strings.ToLower(name)]
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

// Classification diz o que o cofre pensa de um caminho relativo, sem tocar no
// disco. E a mesma decisao que Walk aplica durante a varredura: exporta-la e o
// que impede o watcher de manter uma segunda copia das regras, que divergiria
// da primeira no dia em que alguem acrescentasse uma extensao a so uma delas.
// Classification indica o tipo de arquivo ou se ele deve ser excluido.
type Classification int

const (
	// ClassExcluded indica diretorio excluido, ruido, ou extensao desconhecida.
	ClassExcluded Classification = iota
	// ClassNote indica nota Markdown (.md).
	ClassNote
	// ClassAsset indica arquivo de anexo.
	ClassAsset
)

// Classify classifica um caminho canônico.
func Classify(rel CanonicalPath) Classification {
	parts := strings.Split(string(rel), "/")
	for _, part := range parts {
		if excludedDirs[strings.ToLower(part)] {
			return ClassExcluded
		}
	}

	name := parts[len(parts)-1]
	if isNoise(name) {
		return ClassExcluded
	}

	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".md" {
		return ClassNote
	}
	if assetExts[ext] {
		return ClassAsset
	}
	return ClassExcluded
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
				v.RecordSkip(abs, err)
				return fs.SkipDir
			}
			v.RecordSkip(abs, err)
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

		canon, cErr := Canonicalize(v.walkRoot, abs)
		if cErr != nil {
			v.RecordSkip(abs, cErr)
			return nil
		}

		class := Classify(canon)
		if class == ClassExcluded {
			return nil
		}

		isNote := class == ClassNote

		info, iErr := d.Info()
		if iErr != nil {
			v.RecordSkip(abs, iErr)
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
