// Package boot monta o servidor: cofre, indice de metadados (do cache ou
// construido), indice de busca (adotado do cache, retomado ou construido),
// watcher e Service. Existe porque serve, daemon e os subcomandos de CLI
// precisam da mesma montagem, e ela vivia em cmd/gobsidian, onde nenhum
// teste de pacote a alcancava.
//
// Nao importa mcpsrv nem lifecycle: quem monta nao decide como o host
// conversa nem quando encerra.
package boot
