package mcpsrv

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// noteReadAlvo e um item de `paths`: um caminho, e o que sobrepoe os padroes de
// topo SO para ele.
//
// Duas formas na MESMA lista, de proposito:
//
//	["a.md", {"path": "b.md", "heading": "X"}, {"path": "c.md", "offset": 4000}]
//
// A string continua valendo porque e o que todo cliente manda hoje, e a forma
// nova existe porque seis capitulos de uma obra, com seis headings diferentes,
// exigiam seis chamadas. A alternativa rejeitada era um `items:` ao lado de
// `paths`: cria uma TERCEIRA forma de entrada e torna ternaria a exclusao mutua
// entre `path` e `paths`, que hoje e binaria e testada.
//
// Os campos sobreponiveis sao PONTEIROS: {"path":"b.md","max_bytes":0} pede
// explicitamente "sem teto", e e um pedido diferente de {"path":"b.md"}, que
// herda o teto do topo.
type noteReadAlvo struct {
	Path               string  `json:"path" jsonschema:"Caminho da nota."`
	Heading            *string `json:"heading,omitempty" jsonschema:"Sobrepoe o heading do topo para este item."`
	HeadingLevel       *int    `json:"heading_level,omitempty" jsonschema:"Sobrepoe heading_level para este item."`
	BlockID            *string `json:"block_id,omitempty" jsonschema:"Sobrepoe block_id para este item."`
	Offset             *int64  `json:"offset,omitempty" jsonschema:"Sobrepoe offset para este item. Mutuamente exclusivo com heading e block_id no MESMO item."`
	MaxBytes           *int    `json:"max_bytes,omitempty" jsonschema:"Sobrepoe max_bytes para este item. Zero significa sem teto, e e diferente de omitir."`
	IncludeFrontmatter *bool   `json:"include_frontmatter,omitempty" jsonschema:"Sobrepoe include_frontmatter para este item."`
}

// UnmarshalJSON aceita a string nua e o objeto.
//
// A string nao pode passar a exigir objeto: ela e a forma que todo cliente ja
// manda, e o SDK valida a entrada contra o schema ANTES de chamar este metodo —
// e por isso o schema precisa declarar as duas formas, e nao so esta funcao.
func (a *noteReadAlvo) UnmarshalJSON(data []byte) error {
	var caminho string
	if err := json.Unmarshal(data, &caminho); err == nil {
		*a = noteReadAlvo{Path: caminho}
		return nil
	}

	// Alias evita recursao infinita: json.Unmarshal num tipo com UnmarshalJSON
	// chamaria este metodo de novo.
	type alvoCru noteReadAlvo
	var cru alvoCru
	if err := json.Unmarshal(data, &cru); err != nil {
		return fmt.Errorf("item de paths nao e string nem objeto: %w", err)
	}
	*a = noteReadAlvo(cru)
	return nil
}

// schemaDoNoteRead monta o schema de entrada de note_read.
//
// Existe porque a inferencia do jsonschema-go descreve UM tipo Go por
// propriedade, e `paths` aceita duas formas. O schema inferido diria "objeto", e
// o SDK REJEITARIA a lista de strings na validacao, antes do UnmarshalJSON
// rodar — a forma antiga quebraria em silencio no lado do cliente.
//
// A inferencia continua sendo a conta unica de tudo o mais: as tags de
// noteReadInput geram o schema inteiro, e esta funcao remenda UMA propriedade,
// com o item objeto vindo da propria inferencia de noteReadAlvo. Escrever o
// schema inteiro a mao seria a segunda conta que este projeto ja pagou.
func schemaDoNoteRead() (*jsonschema.Schema, error) {
	esquema, err := jsonschema.For[noteReadInput](nil)
	if err != nil {
		return nil, fmt.Errorf("inferindo o schema de note_read: %w", err)
	}
	paths, ok := esquema.Properties["paths"]
	if !ok {
		return nil, fmt.Errorf("schema de note_read sem a propriedade paths")
	}
	itemObjeto := paths.Items
	if itemObjeto == nil {
		return nil, fmt.Errorf("schema de paths sem items")
	}
	paths.Items = &jsonschema.Schema{
		Description: "Caminho (string) ou objeto {path, heading, heading_level, block_id, offset, max_bytes, include_frontmatter} que sobrepoe os campos de topo so para este item.",
		OneOf: []*jsonschema.Schema{
			{Type: "string"},
			itemObjeto,
		},
	}
	return esquema, nil
}
