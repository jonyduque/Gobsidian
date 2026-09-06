// Package vaulttest oferece aos testes de outros pacotes as condicoes de
// ambiente que o produto promete respeitar e que so o sistema operacional
// pode criar: um arquivo ou diretorio que NAO pode ser aberto, um arquivo
// somente-nuvem, e o prazo unico de espera dos testes.
//
// Cada helper PROVA a condicao antes de devolver. Um handle exclusivo que nao
// barra a leitura tornaria vazia toda assercao de "nao abriu" — foi o que
// aconteceu com metade das copias que este pacote substituiu. A contagem
// medida esta em docs/papeis/testador.md, secao "Handle exclusivo".
//
// Importado apenas por arquivos _test.go. Importa vault e mais nada do
// dominio, para nunca fechar ciclo com quem o usa.
package vaulttest
