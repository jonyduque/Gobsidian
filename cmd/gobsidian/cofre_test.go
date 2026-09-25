package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/config"
)

// O pior defeito possivel da Parte J: `--vault Estudo` e `--vault <caminho de
// Estudo>` dando chaves diferentes. Cada chave e um cache e um socket, entao o
// mesmo cofre teria dois daemons gravando dois caches — o incidente de
// 2026-09-08 por outra porta.
func TestNomeECaminhoDoCofreDaoAMesmaChave(t *testing.T) {
	raiz := t.TempDir()
	cofre := filepath.Join(raiz, "Cofres", "Estudo")
	if err := os.MkdirAll(cofre, 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"vaults": map[string]any{"x": map[string]string{"path": cofre}}})
	registro := filepath.Join(raiz, "obsidian.json")
	if err := os.WriteFile(registro, b, 0o644); err != nil {
		t.Fatal(err)
	}
	cwd := filepath.Join(raiz, "cwd")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	porNome, err := carregarConfigCom(config.Flags{VaultPath: "estudo"}, registro, cwd)
	if err != nil {
		t.Fatalf("pelo nome: %v", err)
	}
	porCaminho, err := carregarConfigCom(config.Flags{VaultPath: cofre}, registro, cwd)
	if err != nil {
		t.Fatalf("pelo caminho: %v", err)
	}

	if porNome.VaultPath != porCaminho.VaultPath {
		t.Errorf("VaultPath: nome %q, caminho %q", porNome.VaultPath, porCaminho.VaultPath)
	}
	if a, b := config.VaultKey(porNome.VaultPath), config.VaultKey(porCaminho.VaultPath); a != b {
		t.Errorf("VaultKey: nome %s, caminho %s — dois caches e dois daemons para um cofre", a, b)
	}
	if porNome.CofrePorNome != "estudo" || porCaminho.CofrePorNome != "" {
		t.Errorf("CofrePorNome: nome %q, caminho %q; quer \"estudo\" e vazio", porNome.CofrePorNome, porCaminho.CofrePorNome)
	}
}

// carregarConfig e a conta unica ate config.Load. Seis subcomandos chamavam
// config.Load direto; um sétimo que voltasse a faze-lo aceitaria --vault
// Estudo como <cwd>\Estudo, em silencio.
func TestSoCarregarConfigChamaConfigLoad(t *testing.T) {
	arquivos, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	vistos := 0
	for _, a := range arquivos {
		if strings.HasSuffix(a, "_test.go") {
			continue
		}
		vistos++
		b, err := os.ReadFile(a)
		if err != nil {
			t.Fatal(err)
		}
		if a != "cofre.go" && strings.Contains(string(b), "config.Load(") {
			t.Errorf("%s chama config.Load direto; use carregarConfig", a)
		}
	}
	if vistos < 6 {
		t.Fatalf("so %d arquivos de producao vistos: o glob nao esta olhando o pacote", vistos)
	}
}
