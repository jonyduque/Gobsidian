### Task 55: `internal/writer/atomic.go` — e o gate do RNF-11

**Esta é a tarefa que bloqueia o marco.** Zero notas corrompidas em 1.000 iterações de crash injetado.

#### O que implementar

Temporário **no mesmo diretório** do alvo — não em `%TEMP%`, porque `rename` entre volumes não é atômico e o Windows o transforma em copiar-e-apagar. Prefixo `.gobsidian-tmp-`, que o filtro de ruído do `vault` já ignora. Então: escrever, **`Sync`**, fechar, `rename` sobre o alvo.

**`Sync` não é opcional e é a linha que alguém remove por parecer lenta.** Sem ele, o `rename` pode ser visível antes de os dados chegarem ao disco, e um corte de energia deixa um arquivo de tamanho certo cheio de zeros. O teste de crash injetado por matar o processo **não** pega isso — só corte de energia real pega. Escreva no comentário que a garantia contra queda de energia depende do `Sync` e que o teste cobre só a metade do processo morto.

**`rename` no Windows falha com o arquivo aberto por outro processo** — antivírus, indexador do Windows, o próprio Obsidian. Precisa de retry com espera curta. `docs/WINDOWS.md` tem o contexto. Retry infinito é travamento; escolha um limite, e devolva erro claro quando estourar.

#### O teste que bloqueia o marco

O crash tem de ser **real**: um subprocesso que escreve e é morto em ponto aleatório, 1.000 vezes. Matar goroutine não é crash — o `defer` roda.

```go
// TestRNF11NoCorruptionUnder1000Crashes e o critério de bloqueio do M4.
// O crash e um PROCESSO morto, nao uma goroutine cancelada: goroutine
// cancelada roda os defers, e sao justamente os defers que o crash tira.
//
// Depois de cada iteracao, o alvo tem de estar em UM de dois estados:
// exatamente o conteudo antigo, ou exatamente o novo. Qualquer terceiro
// estado — truncado, misturado, tamanho certo com zeros — e corrupcao.
func TestRNF11NoCorruptionUnder1000Crashes(t *testing.T) {
	if testing.Short() {
		t.Skip("1000 iteracoes; roda no gate, nao no ciclo curto")
	}
	const iteracoes = 1000
	antigo := []byte("# Antes\r\n\r\nconteudo original preservado\r\n")
	novo := []byte("# Depois\r\n\r\nconteudo novo completo\r\n")

	corrompidas := 0
	for i := 0; i < iteracoes; i++ {
		dir := t.TempDir()
		alvo := filepath.Join(dir, "nota.md")
		if err := os.WriteFile(alvo, antigo, 0o644); err != nil {
			t.Fatal(err)
		}

		// O helper e um subprocesso deste mesmo binario de teste, morto em
		// ponto pseudoaleatorio derivado de i — nao de rand, para a rodada ser
		// reproduzivel a partir do numero da iteracao que falhou.
		matarEm := time.Duration(i%40) * time.Millisecond
		rodarEscritorEMatar(t, alvo, novo, matarEm)

		lido, err := os.ReadFile(alvo)
		if err != nil {
			t.Fatalf("iteracao %d: alvo desapareceu: %v", i, err)
		}
		if !bytes.Equal(lido, antigo) && !bytes.Equal(lido, novo) {
			corrompidas++
			t.Errorf("iteracao %d: estado intermediario (%d bytes): %q",
				i, len(lido), string(lido[:min(80, len(lido))]))
		}
		// Nenhum temporario pode sobrar: ele viraria lixo no cofre do usuario,
		// e o filtro de ruido do vault o esconde da varredura mas nao do disco.
		sobras, _ := filepath.Glob(filepath.Join(dir, ".gobsidian-tmp-*"))
		if len(sobras) > 0 {
			t.Errorf("iteracao %d: temporario sobrou: %v", i, sobras)
		}
	}
	if corrompidas > 0 {
		t.Fatalf("RNF-11 REPROVADO: %d de %d iteracoes corromperam a nota", corrompidas, iteracoes)
	}
	t.Logf("RNF-11: 0 corrompidas em %d iteracoes", iteracoes)
}
```

`rodarEscritorEMatar` usa o padrão de subprocesso de `TestMain` com variável de ambiente — o mesmo que `internal/lifecycle` já usa no harness de órfãos. Reaproveite o padrão de lá; não invente outro.

#### Verificações além dos passos

- **1.000 iterações, zero corrompidas.** Cole a saída inteira. Sem ela, o marco não fecha.
- Nenhum `.gobsidian-tmp-*` sobra ao fim de nenhuma iteração?
- O temporário fica no **mesmo diretório**? Confirme o caminho real, não a intenção.
- `rename` com o alvo aberto por outro processo: o retry funciona? Abra o arquivo num segundo handle e tente.
- CRLF preservado byte a byte? BOM preservado byte a byte? Compare bytes, não strings.
- Escrita em disco cheio devolve erro e deixa o original intacto?

**Prova de mutação obrigatória, três vezes:** remova o `Sync`; troque o temporário para `os.TempDir()`; remova o retry do `rename`. Para cada uma, confirme que um teste nomeado reprova. Se a remoção do `Sync` **não** reprovar nenhum teste, diga isso no relatório — é a lacuna honesta descrita acima, não um teste a inventar.

**Files:** Create `internal/writer/atomic.go`, `atomic_test.go`
**Commit:** `feat(writer): atomic write with fsync and rename retry`

---

