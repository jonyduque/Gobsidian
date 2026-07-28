package index

import (
	"strings"

	"github.com/jonyd/gobsidian/internal/parser"
)

func (ix *Index) resolveAnchor(link *ResolvedLink) {
	if link.Anchor == "" {
		return
	}

	targetNote, ok := ix.notes[link.Resolved]
	if !ok {
		return
	}

	if strings.HasPrefix(link.Anchor, "^") {
		// Block
		targetBlock := link.Anchor[1:]
		for _, b := range targetNote.Blocks {
			if b.ID == targetBlock {
				return
			}
		}
	} else {
		// Heading
		targetSlug := parser.Slug(link.Anchor)
		for _, h := range targetNote.Headings {
			if parser.Slug(h.Text) == targetSlug {
				return
			}
		}
	}

	link.State = LinkAnchorMissing
}
