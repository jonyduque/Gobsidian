package mcpsrv

// MaxPathsPorLote expoe o teto de note_read para os testes de fora do pacote.
// O par de testes do teto (50 aceito, 51 recusado) tem de ser escrito a partir
// da MESMA constante que o produto compara: um literal 50 no teste continua
// verde se alguem mudar o valor do produto, e o par deixa de prender o limite.
// Arquivo _test.go: nada disso existe no binario.
const MaxPathsPorLote = maxPathsPorLote
