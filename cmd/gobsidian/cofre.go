package main

import (
	"os"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/instalar"
)

// carregarConfig e o unico caminho de cmd/gobsidian ate config.Load: resolve
// --vault dado pelo nome do cofre (instalar.ResolverCofre) e entrega a Load o
// caminho. Os seis subcomandos que abrem um cofre passam por aqui, e um teste
// recusa config.Load chamado de outro lugar -- seis copias da resolucao eram
// seis chances de uma delas esquecer.
//
// Tudo abaixo continua derivando do CAMINHO: VaultKey, cache, socket,
// presenca, e o --vault que o spawn repassa ao daemon. Nome e caminho do
// mesmo cofre dao a mesma chave, e e isso que impede dois daemons para um
// cofre.
func carregarConfig(f config.Flags) (config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	return carregarConfigCom(f, instalar.CaminhoDoRegistroDoObsidian(), cwd)
}

func carregarConfigCom(f config.Flags, registro, cwd string) (config.Config, error) {
	caminho, porNome, err := instalar.ResolverCofre(f.VaultPath, registro, cwd)
	if err != nil {
		return config.Config{}, err
	}
	nome := ""
	if porNome {
		nome = f.VaultPath
		f.VaultPath = caminho
	}
	cfg, err := config.Load(f)
	if err != nil {
		return config.Config{}, err
	}
	cfg.CofrePorNome = nome
	return cfg, nil
}
