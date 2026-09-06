package mcpsrv

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// tokenSnakeCase casa qualquer sequencia de letras minusculas e "_", que e a
// forma dos sete valores de service.CamposDeMetadata ("inline_fields" inclui).
// Separadores como ": ", ", " e ";" nunca casam, entao cada match e um
// candidato a valor do enum, nao um pedaco de prosa.
var tokenSnakeCase = regexp.MustCompile(`[a-z_]+`)

// TestNoteMetadataIncludeSchemaListaExatamenteOsValoresAceitos e o achado N1
// da revisao da Task 178: `service.CamposDeMetadata` (a lista que ValidarEnum
// usa) e a tag `jsonschema` de noteMetadataInput.Include (a lista que o host
// le) citavam os mesmos sete valores, mas nada testava que continuassem
// iguais — a tag e texto livre, escrita a mao, sem qualquer ligacao em tempo
// de compilacao com a slice Go.
//
// A tag nao pode ser GERADA a partir de service.CamposDeMetadata: o SDK de
// jsonschema le a struct tag em tempo de COMPILACAO (reflect sobre o tipo),
// entao nao ha como a tag ser "camposDeMetadata, join(", ")" sem inverter a
// dependencia (mcpsrv passaria a construir o texto que o proprio pacote
// declara estaticamente). O teste e o unico jeito de manter as duas listas
// amarradas: ele falha se qualquer um dos dois lados ganhar, perder ou trocar
// um valor sem o outro acompanhar.
func TestNoteMetadataIncludeSchemaListaExatamenteOsValoresAceitos(t *testing.T) {
	campo, ok := reflect.TypeOf(noteMetadataInput{}).FieldByName("Include")
	if !ok {
		t.Fatal("noteMetadataInput nao tem mais um campo Include")
	}
	tag := campo.Tag.Get("jsonschema")
	if tag == "" {
		t.Fatal("Include perdeu a tag jsonschema")
	}

	idxAceitos := strings.Index(tag, "aceitos:")
	if idxAceitos == -1 {
		t.Fatalf("tag nao tem mais o marcador \"aceitos:\": %q", tag)
	}
	idxFim := strings.Index(tag[idxAceitos:], ";")
	if idxFim == -1 {
		t.Fatalf("tag nao fecha a lista de aceitos com \";\": %q", tag)
	}
	trechoDaLista := tag[idxAceitos : idxAceitos+idxFim]

	aceitos := map[string]bool{}
	for _, tok := range tokenSnakeCase.FindAllString(trechoDaLista, -1) {
		if tok == "aceitos" {
			continue
		}
		aceitos[tok] = true
	}

	querido := map[string]bool{}
	for _, v := range service.CamposDeMetadata {
		querido[v] = true
	}

	// Cada valor de service.CamposDeMetadata precisa aparecer, como token
	// inteiro, dentro do trecho "aceitos: ...;". Um oitavo valor acrescentado
	// SO em CamposDeMetadata (e nao na tag) reprova aqui.
	for v := range querido {
		if !aceitos[v] {
			t.Errorf("service.CamposDeMetadata tem %q, mas a tag jsonschema de Include nao lista: %q", v, tag)
		}
	}
	// E nenhum token dentro do trecho de aceitos pode ser estranho a
	// service.CamposDeMetadata. Um oitavo valor acrescentado SO na tag (e nao
	// em CamposDeMetadata) reprova aqui.
	for tok := range aceitos {
		if !querido[tok] {
			t.Errorf("a tag jsonschema de Include lista %q, que nao esta em service.CamposDeMetadata: %q", tok, tag)
		}
	}
}
