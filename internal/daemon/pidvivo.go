package daemon

// PIDVivo diz se o processo existe. E um invólucro exportado sobre a
// implementação por plataforma (pidvivo_windows.go, pidvivo_unix.go), e não uma
// segunda conta do mesmo fato: quem precisa da resposta fora deste pacote —
// hoje o `doctor`, ao acusar lock órfão — chama esta, e a lógica continua
// morando num lugar só.
//
// A alternativa seria o doctor reimplementar a checagem, e a lição de `byAlias`
// já custou caro o suficiente para não repetir o padrão: duas contas do mesmo
// fato concordam até o dia em que uma delas muda sozinha. A armadilha aqui é
// concreta e está documentada em pidvivo_windows.go — o Windows mantém PID e
// creation time consultáveis muito depois da morte do processo, e uma segunda
// implementação ingênua responderia "vivo" para todo lock antigo.
func PIDVivo(pid int) bool { return pidVivo(pid) }
