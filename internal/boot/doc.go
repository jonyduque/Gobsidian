// Package boot monta o servidor: cofre, indice de metadados (do cache ou
// construido), indice de busca (adotado do cache, retomado ou construido),
// watcher e Service. Existe porque serve e daemon precisam da mesma
// montagem, e ela vivia em cmd/gobsidian, onde nenhum teste de pacote a
// alcancava. Os subcomandos de CLI abrem o indice por aqui — index, inspect
// e search chamam AbrirIndice, e search tambem chama PrepararBusca —, mas
// nao chegam a Montar: nao constroem watcher nem Service.
//
// Monta tambem a VIGILIA do host (VigiarHost) e os passos de encerramento
// que os tres pontos de saida repetiam — o pipe espelhado mais lifecycle.New
// em serve e na ponte, e os passos "close-pipe" e "watcher". Por isso importa
// lifecycle: e o mesmo andaime, e a ordem entre as pecas importa.
//
// Nao importa mcpsrv: quem monta nao decide como o host conversa. E nao
// decide QUANDO encerrar — lifecycle.Shutdown continua sendo chamada por
// quem serve, com os passos que so ela conhece (in-flight, half-close,
// close-conn).
package boot
