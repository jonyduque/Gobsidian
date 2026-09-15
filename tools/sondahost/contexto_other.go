//go:build !windows

package main

// contexto so e medido no Windows, onde o defeito da Parte G foi encontrado.
// Nas outras plataformas a linha diz isso, em vez de fingir um contexto.
func contexto() string {
	return "contexto nao verificado nesta plataforma"
}
