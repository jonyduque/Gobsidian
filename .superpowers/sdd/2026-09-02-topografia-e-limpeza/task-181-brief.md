### Task 181: c2 — testes de caixa-branca do codec do cache de metadados

**Files:**
- Create: `internal/index/persist_codec_test.go` (pacote `index`)
- Modify: `docs/ESTADO.md` (dívida 1.12 fechada; cobertura de `persist_codec.go` antes/depois)

**Interfaces:**
- Consumes (todos não exportados, `internal/index/persist_codec.go`): tags `valNil, valBool, valInt, valInt64, valUint64, valFloat64, valString, valTime, valSliceNil, valSlice, valMapNil, valMap`; limites `limiteString`, `limiteValorProfund`; `escritor{w io.Writer}` com métodos `value(any)`, `note(*Note)`, `str`, `strSlice`, `timeBlob`; `leitor{b []byte}` com `value(profundidade int)`, `note()`, `strSlice()`, `timeBlob()` e campo `err`; `ErrIndexCacheCorrupted`. Nomes e aridades exatas: `gopls` sobre `persist_codec.go` — os esboços abaixo usam os nomes do arquivo em 2026-09-02; se um método receber argumento a mais (`leitor.str(oque string)`), passar a string.
- Produces: nada — fecha a dívida 1.12 de `ESTADO.md`.

- [ ] **Step 1: Cobertura antes**

Run: `go test ./internal/index/ -coverprofile=%TEMP%\cov_antes.out && go tool cover -func=%TEMP%\cov_antes.out | grep persist_codec.go` — colar todas as linhas (uma por função).

- [ ] **Step 2: Os testes**

`internal/index/persist_codec_test.go`:

```go
package index

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func roundTrip(t *testing.T, v any) any {
	t.Helper()
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.value(v)
	if e.err != nil {
		t.Fatalf("escrevendo %#v: %v", v, e.err)
	}
	l := &leitor{b: buf.Bytes()}
	got := l.value(0)
	if l.err != nil {
		t.Fatalf("lendo %#v: %v", v, l.err)
	}
	if l.i != len(l.b) {
		t.Fatalf("sobraram %d bytes depois de ler %#v", len(l.b)-l.i, v)
	}
	return got
}

func TestCodecValorRoundTripPorTipo(t *testing.T) {
	zona := time.FixedZone("X", -3*3600)
	casos := []any{
		nil, true, false, int(-7), int64(1 << 40), uint64(1<<63 + 5), float64(2.5),
		"ação", "", time.Date(2026, 9, 2, 10, 0, 0, 123, zona),
		[]any{int(1), "dois", nil}, []any{}, map[string]any{"k": "v", "n": int(3)}, map[string]any{},
		[]any{map[string]any{"a": []any{int(1)}}},
	}
	for _, in := range casos {
		got := roundTrip(t, in)
		if in == nil {
			if got != nil {
				t.Errorf("nil virou %#v", got)
			}
			continue
		}
		if reflect.TypeOf(got) != reflect.TypeOf(in) {
			t.Errorf("%#v (%T) voltou como %T", in, in, got)
			continue
		}
		if tm, ok := in.(time.Time); ok {
			if !tm.Equal(got.(time.Time)) {
				t.Errorf("time %v voltou %v", tm, got)
			}
			continue
		}
		if !reflect.DeepEqual(got, in) {
			t.Errorf("%#v voltou %#v", in, got)
		}
	}
}

func TestCodecStrSliceNilDistintoDeVazio(t *testing.T) {
	for _, in := range [][]string{nil, {}} {
		var buf bytes.Buffer
		e := &escritor{w: &buf}
		e.strSlice(in)
		l := &leitor{b: buf.Bytes()}
		got := l.strSlice()
		if l.err != nil {
			t.Fatal(l.err)
		}
		if (got == nil) != (in == nil) {
			t.Errorf("strSlice %#v voltou %#v (nil-ness diferente)", in, got)
		}
	}
}

func TestCodecTimeBlobPreservaZona(t *testing.T) {
	in := time.Date(2026, 9, 2, 10, 0, 0, 7, time.FixedZone("Y", 5*3600+1800))
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.timeBlob(in)
	l := &leitor{b: buf.Bytes()}
	got := l.timeBlob()
	if l.err != nil {
		t.Fatal(l.err)
	}
	if !got.Equal(in) {
		t.Fatalf("%v voltou %v", in, got)
	}
	_, off1 := in.Zone()
	_, off2 := got.Zone()
	if off1 != off2 {
		t.Fatalf("offset de zona %d voltou %d", off1, off2)
	}
}

func TestCodecValorTipoNaoSuportadoFalha(t *testing.T) {
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.value(struct{}{})
	if e.err == nil || !strings.Contains(e.err.Error(), "tipo nao suportado") {
		t.Fatalf("struct{}{} devia falhar com \"tipo nao suportado\", tenho %v", e.err)
	}
}

func TestCodecValorProfundidadeAlemDoLimiteERecusada(t *testing.T) {
	var v any = "folha"
	for i := 0; i < limiteValorProfund+6; i++ {
		v = []any{v}
	}
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.value(v)
	if e.err == nil {
		// O escritor pode nao limitar profundidade; entao o leitor tem de limitar.
		l := &leitor{b: buf.Bytes()}
		_ = l.value(0)
		if l.err == nil || !errors.Is(l.err, ErrIndexCacheCorrupted) || !strings.Contains(l.err.Error(), "profundidade") {
			t.Fatalf("profundidade %d devia ser recusada na leitura, tenho %v", limiteValorProfund+6, l.err)
		}
		return
	}
	if !strings.Contains(e.err.Error(), "profundidade") {
		t.Fatalf("escritor recusou por outro motivo: %v", e.err)
	}
}

func TestCodecTagDesconhecidaERecusada(t *testing.T) {
	l := &leitor{b: []byte{valMap + 1}}
	_ = l.value(0)
	if l.err == nil || !errors.Is(l.err, ErrIndexCacheCorrupted) || !strings.Contains(l.err.Error(), "tag de valor desconhecida") {
		t.Fatalf("tag %d devia ser recusada, tenho %v", valMap+1, l.err)
	}
}

func TestCodecStringAcimaDoLimiteERecusada(t *testing.T) {
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.uvarint(uint64(limiteString + 1))
	l := &leitor{b: buf.Bytes()}
	_ = l.str("teste")
	if l.err == nil || !strings.Contains(l.err.Error(), "acima do limite") {
		t.Fatalf("string de %d bytes devia ser recusada, tenho %v", limiteString+1, l.err)
	}
}

func TestCodecNotaTruncadaEmCadaByteERecusada(t *testing.T) {
	n := &Note{
		Path: "pasta/nota.md", Title: "Nota", Hash: "abc",
		Tags: []string{"a", "b/c"}, Aliases: []string{"x"},
		Frontmatter: map[string]any{"k": "v", "n": int64(2), "lista": []any{"a", int(1)}},
		ModTime: time.Now(),
	}
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.note(n)
	if e.err != nil {
		t.Fatal(e.err)
	}
	b := buf.Bytes()
	for i := 0; i < len(b); i++ {
		l := &leitor{b: b[:i]}
		_ = l.note()
		if l.err == nil {
			t.Fatalf("prefixo de %d/%d bytes foi aceito como nota completa", i, len(b))
		}
	}
	l := &leitor{b: b}
	got := l.note()
	if l.err != nil || got == nil || got.Path != n.Path {
		t.Fatalf("a nota inteira devia ler: err=%v got=%+v", l.err, got)
	}
}
```

Os campos de `Note` usados (`Path`, `Title`, `Hash`, `Tags`, `Aliases`, `Frontmatter`, `ModTime`) são os de `note.go`; preencher também `Headings`, `Blocks`, `Links`, `InlineFields` com um elemento cada, para que o truncamento percorra os quatro sub-codecs — os tipos exatos estão em `note.go` (`gopls`). Se `escritor.value` não devolver erro para profundidade (só o leitor limita), o teste de profundidade já cobre os dois caminhos.

Run: `go test -race ./internal/index/ -run TestCodec -v` — Expected: PASS em todos. Um FAIL aqui é achado real (o codec aceita o que não devia) — reportar como `DONE_WITH_CONCERNS` com o teste **mantido** e a linha do codec apontada; não afrouxar o teste.

- [ ] **Step 3: Provas de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'if profundidade > limiteValorProfund {' -Replacement 'if false {' -Test TestCodecValorProfundidadeAlemDoLimiteERecusada -Package ./internal/index/
pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'case valInt64:' -Replacement 'case valInt64 + 100:' -Test TestCodecValorRoundTripPorTipo -Package ./internal/index/
```

Expected: exit 0 nos dois. A grafia exata das âncoras é a do arquivo (`grep -n "limiteValorProfund\|case valInt64" internal/index/persist_codec.go`); ajustar a âncora ao texto real, não a mutação.

- [ ] **Step 4: Cobertura depois e `ESTADO.md`**

Run: `go test ./internal/index/ -coverprofile=%TEMP%\cov_depois.out && go tool cover -func=%TEMP%\cov_depois.out | grep persist_codec.go` — colar. `docs/ESTADO.md`: na dívida 1.12, "fechada em 2026-09 (Task 181): testes de caixa-branca em `persist_codec_test.go`; cobertura de `persist_codec.go` X % → Y %" com os dois números **medidos** no Step 1 e aqui (o total do arquivo é a média ponderada que `go tool cover -func` não dá direto — publicar por função, as linhas coladas, e não inventar uma média).

- [ ] **Step 5: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/index/persist_codec_test.go docs/ESTADO.md
git commit -m "test(index): white-box tests for the metadata cache codec"
```

#### Verificações

- Cobertura antes (Step 1) e depois (Step 4) coladas, por função.
- PASS do Step 2 colado; dois `mutate.ps1` exit 0 colados.
- Nenhum arquivo de produto no commit: `git show --stat HEAD` só com `persist_codec_test.go` e `ESTADO.md`.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Só teste e doc: se um teste revelar defeito no codec, o defeito vai para o relatório e para `SUGESTOES.md`; corrigir é outra Task.
- Nenhum número de cobertura sem a linha do `go tool cover` colada ao lado.

#### Comando de mutação

Step 3 (dois `mutate.ps1` sobre `internal/index/persist_codec.go`).

#### Contrato de relatório

`task-181-report.md`: status, SHA, cobertura antes/depois, saída dos Steps 2 e 3, `git show --stat HEAD`, última linha do `verify.ps1`.

