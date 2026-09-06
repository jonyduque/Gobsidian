// Package boot monta o servidor: cofre, indice de metadados (do cache ou
// construido), indice de busca (adotado do cache, retomado ou construido),
// watcher e Service. Existe porque serve e daemon precisam da mesma
// montagem, e ela vivia em cmd/gobsidian, onde nenhum teste de pacote a
// alcancava. Os subcomandos de CLI nao passam por aqui: index, search e
// inspect montam o indice por conta propria, sem watcher e sem Service.
//
// Nao importa mcpsrv nem lifecycle: quem monta nao decide como o host
// conversa nem quando encerra.
package boot
