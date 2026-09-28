// Package textos guarda, num lugar so, tudo o que os comandos de CLI escrevem
// na tela.
//
// # Por que um arquivo so
//
// Ate 2026-09-16 cada mensagem morava ao lado do codigo que a imprimia. O
// efeito pratico: a saida do produto misturava portugues correto com portugues
// sem acento -- "permissão de leitura" numa linha e "caminho da raiz do cofre
// (obrigatorio)" na ajuda da flag logo abaixo --, e nao havia como revisar a
// redacao sem percorrer catorze arquivos. Aqui da para ler tudo de uma vez.
//
// A redacao e revista pelo dono em docs/TEXTOS.md, que mostra cada constante
// como ela sai na tela. As regras de redacao estao la: frase e titulo com
// maiuscula e ponto final, ponto e virgula trocado por ponto, flag e comando
// entre crases, termo tecnico em ingles em italico.
//
// # Por que constantes Go, e nao JSON ou .txt embutido
//
// Quase todo texto daqui carrega verbo de formatacao (%s, %d, %q). Constante
// Go deixa `go vet` conferir que o verbo casa com o argumento no ponto de
// chamada -- um catalogo carregado em tempo de execucao perde essa conferencia
// e ainda acrescenta um modo de falha novo, a chave que nao existe. O texto do
// servidor MCP (internal/mcpsrv/instrucoes.txt) e embutido porque e um bloco
// unico sem formatacao; este e o caso oposto.
//
// # UTF-8 por padrao
//
// Tudo aqui e escrito em portugues correto, com acento. Quem decide se o
// acento chega a tela e console.adaptarTexto, pela code page do console no
// Windows e pelo locale no resto -- onde nao aguenta, o acento cai na hora de
// imprimir. Escrever "instalacao" no codigo para agradar um CP-850 e o que
// fazia o produto inteiro falar errado.
//
// # Marcacao
//
// **negrito**, *italico* e `codigo` ficam no texto como o dono os escreveu, e
// quem os desenha e o console (console.marcar): com cor, viram realce; sem cor,
// negrito e italico somem e as crases ficam. Texto que o shell mostra sozinho
// -- a descricao ao lado de um valor no autocompletar -- nao leva marcacao,
// porque ali ninguem a desenha.
//
// # Erro com maiuscula e ponto
//
// Os erros daqui sao frases que o usuario le, e seguem a regra do documento --
// o que contraria a convencao de Go para string de erro. A excecao vale so
// para este pacote: erro que embrulha outro (%w ou %v no fim) nao leva ponto,
// porque o de dentro traz o dele.
//
// # O que NAO mora aqui
//
// Nome de comando e de flag (`install`, `--vault`), chave de host e valor de
// enum: sao interface de linha de comando, nao prosa -- mudam o contrato com
// quem automatiza, e nao a leitura de quem opera. Mensagem de log tambem nao:
// log sai por slog, em ingles de maquina, e nao passa pelo console.
package textos

// Ajuda dos comandos: o `Short` e o `Long` que o cobra mostra, e as partes do
// template de ajuda.
const (
	ResumoRaiz = "Servidor MCP para cofres locais do Obsidian."

	ResumoServe      = "Serve o cofre via MCP sobre stdio."
	ResumoDaemon     = "Roda o *daemon* de cofre compartilhado (uso interno da ponte)."
	ResumoDoctor     = "Diagnostica o ambiente."
	ResumoIndex      = "Constrói o índice do cofre e exibe um resumo."
	ResumoInspect    = "Exibe metadados, links e *backlinks* de uma nota."
	ResumoSearch     = "Busca um texto no cofre."
	ResumoVersion    = "Imprime versão."
	ResumoHelp       = "Exibe a ajuda sobre qualquer comando."
	ResumoCompletion = "Gera o script de autocompletar do shell."

	ResumoInstall    = "Instala o **gobsidian**."
	DescricaoInstall = "Instala o executável no perfil do usuário, sem elevação, pergunta se deve " +
		"acrescentar o diretório ao *PATH* e registrar o servidor nos hosts de IA detectados."

	ResumoPath   = "Acrescenta ou remove o diretório de instalação do *PATH* do usuário."
	ResumoConfig = "Configura o **gobsidian** como MCP de hosts de IA."
	ResumoVaults = "Lista os cofres do Obsidian e os configurados nos hosts de IA."

	ResumoUpdate    = "Atualiza o **gobsidian**."
	DescricaoUpdate = "Consulta a versão publicada, baixa o binário da plataforma corrente, " +
		"confere o SHA-256 publicado e só então encerra os processos em execução " +
		"e troca o binário. Divergência de soma aborta sem instalar nada."

	AjudaUso      = "Uso:"
	AjudaComandos = "Comandos disponíveis:"
	AjudaFlags    = "Flags:"
	AjudaDetalhes = "Use `%s [comando] --help` para detalhes de um comando."
	FlagHelp      = "Exibe a ajuda do **%s**."
)

// Ajuda das flags. O nome da flag fica no codigo; o que muda aqui e a frase.
const (
	FlagVault          = "Nome do cofre no Obsidian ou caminho da raiz (obrigatório)."
	FlagFollowSymlinks = "Segue *symlink* dentro do cofre."
	FlagCacheDir       = "Diretório do cache de índice."
	FlagLogLevel       = "Nível de log: `debug`, `info`, `warn` ou `error`."
	FlagJSON           = "Força a saída em JSON, mesmo no terminal."
	FlagTexto          = "Força a saída para ler, mesmo fora do terminal."
	FlagReadOnly       = "Desabilita toda a superfície de escrita."
	FlagDebounce       = "Janela de coalescência de eventos do *watcher*."
	FlagMaxResults     = "Teto de resultados por consulta."
	FlagEagerSearch    = "Carrega o índice de busca no *boot* em vez de esperar a primeira `vault_search`."
	FlagIdleSeconds    = "Segundos sem cliente conectado antes de o *daemon* encerrar (padrão: 15 minutos)."

	FlagDoctorReadOnly = "Não verifica permissão de escrita."
	FlagDoctorFix      = "Além de diagnosticar, remove o lixo de *runtime* e do cache."

	FlagSearchLimit = "Limite máximo de resultados."

	FlagInstallVault    = "Cofre a servir (padrão: listar cofres registrados no Obsidian para escolha)."
	FlagInstallDir      = "Diretório onde o **gobsidian** será instalado (padrão: `%s`)."
	FlagInstallHosts    = "Lista de hosts de IA (Claude, Codex, AGY, etc.) a configurar, separados por vírgula (`%s`). `none` não configura nenhum."
	FlagInstallYes      = "Não pergunta nada: instala, ajusta o *PATH* e configura todos os hosts de IA detectados."
	FlagInstallReadOnly = "Registra o servidor apenas para leitura."
	FlagInstallNoPath   = "Não altera o *PATH*."

	FlagPathAdd    = "Acrescenta o diretório ao *PATH*."
	FlagPathRemove = "Remove o diretório do *PATH*."

	FlagUpdateCheck = "Só diz se há versão nova, sem baixar nem instalar."
	FlagUpdateYes   = "Não pergunta antes de encerrar os processos em execução."
)

// Erros que o usuario le no terminal.
const (
	ErroIdleSeconds        = "`--idle-seconds` precisa ser >= 1 (recebido %d)."
	ErroResolvendoNota     = "Resolvendo nota %q: %w"
	ErroNotaNaoIndexada    = "Nota %q não encontrada no índice."
	ErroSemCofre           = "Nenhum cofre encontrado. Passe `--vault` com o caminho do cofre."
	ErroSelecaoCancelada   = "Seleção cancelada. Nada foi alterado."
	ErroHostDesconhecido   = "Host desconhecido %q. Conhecidos: %s"
	ErroEscolhaInvalida    = "Escolha inválida: %q"
	ErroEscolhaForaDaLista = "Escolha fora da lista: %d"
	ErroAddOuRemove        = "Escolha exatamente um: `--add` ou `--remove`."
	ErroSemManifesto       = "%w -- Rode `gobsidian install` primeiro."
	ErroManifestoAusente   = "Não há manifesto de instalação"
	ErroConsultarRelease   = "Consultando *releases*: %w"
	ErroAtivoAusente       = "O *release* %s não publica %q (plataforma %s/%s)."
	ErroTemporario         = "Criando diretório temporário: %w"
	ErroGerandoScript      = "Gerando o script de %s: %w"
	ErroJSONETexto         = "`--json` e `--texto` não podem ser usados juntos."
)

// Erros de --vault dado pelo nome do cofre (instalar.ResolverCofre). Cada um
// termina dizendo o que fazer, porque quem le e quem configurou o host.
const (
	ErroCofreAmbiguo      = "O nome %q casa mais de um cofre do Obsidian: %s. Passe o caminho da raiz em `--vault`."
	ErroCofreNomeEPasta   = "O nome %q é o cofre %s do Obsidian e também a pasta %s no diretório atual. Passe o caminho da raiz em `--vault`."
	ErroCofreSemRegistro  = "%q não é uma pasta e o Obsidian não tem registro de cofres (%s). Passe o caminho da raiz do cofre em `--vault`."
	ErroCofreDesconhecido = "Nenhum cofre do Obsidian se chama %q. Conhecidos: %s. Passe o nome de um deles ou o caminho da raiz em `--vault`."
	CofreNenhumConhecido  = "nenhum"
)

// `gobsidian` sem argumento nenhum: a autoinstalacao da decisao D-11.
const (
	AutoinstalarNaoInstalado = "O **gobsidian** ainda não está instalado nesta máquina."
	AutoinstalarDestino      = "Vamos instalar em `%s`."
	AutoinstalarAjuda        = "Para ver apenas a ajuda, rode `gobsidian --help`."
)

// `version`, `index`, `inspect` e `search`.
const (
	VersionTitulo = "**gobsidian** %s"
	CampoCommit   = "*Commit*"
	CampoBuild    = "*Build*"

	IndexConcluido  = "Indexação concluída em %d ms."
	IndexTitulo     = "Índice."
	CampoOrigem     = "Origem"
	CampoNotas      = "Notas"
	CampoAnexos     = "Anexos"
	CampoTags       = "Tags"
	CampoTamanho    = "Tamanho"
	NotaBytes       = "bytes"
	CampoTitulo     = "Título"
	CampoHeadings   = "*Headings*"
	CampoLinksSaida = "Links de saída"
	CampoBacklinks  = "*Backlinks*"

	SearchSemResultado = "Nenhum resultado para %q."
	SearchTitulo       = "%d de %d para %q."
)

// As perguntas da conversa: botoes, teclas e respostas. As teclas e as setas
// sao glifos (console.Glifos); o que mora aqui e a palavra que diz o que a
// tecla faz.
const (
	BotaoSim = "Sim"
	BotaoNao = "Não"

	TeclaMover      = "mover"
	TeclaResponder  = "responder"
	TeclaMarcar     = "marcar"
	TeclaTodos      = "todos"
	TeclaConfirmar  = "confirmar"
	TeclaEspaco     = "espaço"
	TeclaSimNao     = "[s]/[n]"
	TeclaTodosLetra = "[a]"

	SufixoSimPadrao   = "S/n"
	SufixoNaoPadrao   = "s/N"
	RespostaSim       = "Sim"
	RespostaNao       = "Não"
	RespostaCancelado = "Cancelado"
)

// `install`, `vaults` e `path`.
const (
	InstallSemTerminal      = "A entrada não é um terminal interativo, então serão utilizadas as opções padrão."
	InstallSemTerminalDica  = "Para escolher os cofres, rode `gobsidian install` em um terminal interativo."
	InstallTitulo           = "Instalando."
	InstallCancelado        = "Instalação cancelada. Nada foi alterado."
	InstallConcluido        = "Instalado."
	InstallRegistroIlegivel = "Não foi possível ler os cofres do Obsidian: %v"
	InstallCofreEscolhido   = "Cofre: %s"
	InstallConfigAtual      = "Configuração atual."
	InstallManterConfig     = "Manter esta configuração?"
	VaultsMantido           = "Configuração mantida. Nada foi alterado."
	InstallQuaisCofres      = "Quais cofres configurar?"
	InstallQuaisHosts       = "Em quais hosts registrar?"
	InstallSemHosts         = "Nenhum host de IA conhecido foi detectado."
	InstallHostsEncontrados = "Hosts de IA encontrados."
	InstallEscolhaDigitada  = "Números separados por espaço, `*` para todos, vazio para nenhum."
	InstallItemNumerado     = "%d) %s  *%s*"
	InstallItemHost         = "%d) %s  *%s*"
	InstallYesEncerrando    = "`--yes`: encerrando sem perguntar."

	InstallEncerrarProcessos = "Encerrar estes processos do **gobsidian** para trocar o binário?"
	InstallProcessoItem      = "pid %d  %s  %s"

	InstallHostsConfigurados = "Hosts configurados."
	InstallSemCofreNaLista   = "Nenhum cofre configurado."
	InstallCofreNaLista      = "Cofre    %s"
	InstallHostNaLista       = "%-16s %s"
	InstallHostFalhou        = "%s não pode ser configurado."

	ResumoBinario   = "Binário  %s"
	ResumoCofres    = "Cofres   nenhum configurado"
	ResumoCofre     = "Cofre    %s"
	ResumoPATH      = "*PATH*     %s"
	ResumoEncerrado = "Encerrado pid %d (%s)"
	ResumoHostOK    = "%-16s %s"
	ResumoHostFalha = "%-16s FALHOU: %s"
	ResumoLimpeza   = "Limpeza  %s trava(s), %s *socket(s)*, %s presença(s), %s cache(s), %d KB"

	PathJaEstava   = "*PATH* já estava como você pediu."
	PathAtualizado = "*PATH* atualizado."
)

// `vaults`: a lista de cofres.
const (
	VaultsTitulo         = "Cofres."
	VaultsNenhum         = "Nenhum cofre encontrado no Obsidian nem nos hosts de IA."
	VaultsAberto         = "aberto no Obsidian"
	VaultsConfigurado    = "configurado"
	VaultsForaDoObsidian = "fora do Obsidian"
	VaultsRodape         = "Rode `gobsidian config` para registrar um cofre nos hosts de IA."
)

// `update`.
const (
	UpdateConsultando  = "Consultando a última versão publicada."
	UpdateInstalada    = "Versão instalada: %s"
	UpdatePublicada    = "Versão publicada: %s"
	UpdateJaAtual      = "Já está atualizado."
	UpdateHaVersaoNova = "Há versão nova: %s"
	UpdateComoInstalar = "Rode `gobsidian update` para instalar."
	UpdateBaixando     = "Baixando %s e conferindo o SHA-256."
	UpdateHashDiverge  = "O binário baixado NÃO confere com a soma publicada."
	UpdateNadaMudou    = "Nada foi instalado. Sua instalação continua intacta."
	UpdateHashConfere  = "SHA-256 confere."
	UpdateTrocando     = "Trocando o binário."
	UpdateCancelado    = "Atualização cancelada. Nada foi alterado."
	UpdateConcluido    = "Atualizado para %s."
	UpdateHostsSozinho = "Os hosts reiniciam o servidor sozinhos. Não precisa reiniciar manualmente."
)

// `doctor`: o relatorio, os processos e o lixo.
const (
	DoctorTitulo       = "Diagnóstico do ambiente."
	DoctorFalhasAcima  = "Há falhas bloqueantes acima."
	DoctorAmbienteApto = "Ambiente apto."
	DoctorResumo       = "%d verificações: %s, %s, %s"
	// As palavras da contagem so saem onde o console nao aguenta emoji: la,
	// aviso e falha teriam o mesmo marcador "[!]". Onde aguenta, a contagem
	// usa o proprio marcador do estado.
	DoctorResumoOK     = "ok"
	DoctorResumoAviso  = "aviso"
	DoctorResumoAvisos = "avisos"
	DoctorResumoFalha  = "falha"
	DoctorResumoFalhas = "falhas"

	DoctorRuntimeIndisponivel = "Processos e lixo: diretório de *runtime* indisponível (%v)."
	DoctorProcessosErro       = "Processos do **gobsidian**: %v"
	DoctorProcessos           = "Processos do **gobsidian**."
	DoctorNenhumProcesso      = "Nenhum rodando."
	DoctorProcessosContagem   = "%d processo(s) do **gobsidian**."
	DoctorProcessosRodape     = "Um servidor por sessão do host. O *daemon* é um só por cofre."
	DoctorGravadoresDuplos    = "%d processos gravam o cache do mesmo cofre."
	DoctorGravadoresDetalhe   = "%s -- gravadores: %s. Encerre os extras. Pontes não gravam e ficam fora desta conta."
	DoctorSemModoDetalhe      = "%s -- %s sem modo registrado (versão anterior): não dá para saber se gravam."
	DoctorPid                 = "pid %d"
	DoctorPids                = "pid %s"

	DoctorSemPresencaPlataforma = "Processos do **gobsidian** sem presença: não verificado nesta plataforma."
	DoctorSemPresencaErro       = "Processos do **gobsidian** sem presença: %v"
	DoctorSemPresencaNenhum     = "Processos do **gobsidian** sem presença."
	DoctorSemPresencaVazio      = "Nenhum."
	DoctorSemPresencaContagem   = "%d processo(s) do **gobsidian** sem presença."
	DoctorSemPresencaRodape     = "Binário anterior à presença, ou de outra instalação. O `doctor` não sabe o modo nem o cofre deles."

	DoctorHostsSemManifesto = "Binário dos hosts: sem instalação registrada, nada a comparar."
	DoctorHostsErro         = "Binário dos hosts: %v"
	DoctorHostsOK           = "Binário dos hosts."
	DoctorHostsDetalhe      = "Toda entrada do **gobsidian** nos configs de arquivo roda %s."
	DoctorHostsOutroBinario = "%d entrada(s) de host rodam outro binário."
	DoctorHostsRodape       = "O instalado é %s (%s). `gobsidian install` reconfigura. Claude Code, Gemini CLI, Codex e VS Code guardam a configuração no próprio CLI e não entram aqui."
	DoctorVersaoNaoMedida   = "versão não medida"
	DoctorComandoAusente    = "comando não encontrado"

	DoctorChavesErro     = "Chaves de cache: %v"
	DoctorChavesSuperada = "%d cache(s) sob chave superada."
	DoctorChavesRodape   = "Rode `gobsidian update` para renomear com tudo encerrado."

	DoctorLixoErro         = "Lixo do diretório de *runtime*: %v"
	DoctorLixoNenhum       = "Lixo de execuções anteriores."
	DoctorLixoNadaARemover = "Nada a remover."
	DoctorLixoTitulo       = "Lixo de execuções anteriores."
	DoctorLixoRemovivel    = "removível"
	DoctorLixoRemovido     = "removido"
	DoctorLixoTravas       = "Travas"
	DoctorLixoSockets      = "*Sockets*"
	DoctorLixoPresencas    = "Presenças"
	DoctorLixoCaches       = "Caches"
	DoctorLixoCachesNota   = "de cofre inexistente"
	DoctorLixoLogs         = "Logs rotacionados"
	DoctorLixoTotal        = "Total"
	DoctorLixoKB           = "%d KB"
	DoctorLixoComoLimpar   = "Rode `gobsidian doctor --fix` para remover, ou `gobsidian update`, que já limpa."
	DoctorLixoNaoRemovido  = "Não removido: %s"

	DoctorModoDaemon        = "*daemon*"
	DoctorModoDaemons       = "*daemons*"
	DoctorModoPonte         = "ponte"
	DoctorModoPontes        = "pontes"
	DoctorModoEmProcesso    = "servidor em processo"
	DoctorModoEmProcessos   = "servidores em processo"
	DoctorModoNaoRegistrado = "modo não registrado"
	DoctorSemCofre          = "(sem cofre registrado)"
)

// Os grupos e os nomes das verificações do `doctor`: o que abre cada linha do
// relatório.
const (
	GrupoCofre   = "Cofre."
	GrupoCache   = "Cache e disco."
	GrupoDaemon  = "*Daemon*."
	GrupoWindows = "Windows."

	CheckRaizExiste       = "Raiz do cofre existe."
	CheckLeitura          = "Permissão de leitura."
	CheckEscrita          = "Permissão de escrita."
	CheckObsidian         = "`.obsidian` presente."
	CheckNotas            = "Contagem de notas."
	CheckCaminho          = "Comprimento de caminho."
	CheckCache            = "Diretório de cache."
	CheckEspaco           = "Espaço em disco."
	CheckCaminhosLongos   = "Caminhos longos habilitados."
	CheckSomenteNuvem     = "Arquivos somente-nuvem."
	CheckCasing           = "Colisões de *casing*."
	CheckSocket           = "Caminho do *socket* do *daemon*."
	CheckDiretorioSockets = "Diretório de *sockets* aceita conexão."
	CheckDaemonVivo       = "*Daemon* respondendo."
	CheckLogDaemon        = "Log do *daemon*."
	CheckTravas           = "Travas de *daemon* em uso."
)

// Os detalhes de cada verificação: o número ou o caminho que torna o resultado
// acionável.
const (
	DetVarreduraInterrompida = "Varredura interrompida: %v"
	DetCofreInacessivel      = "Cofre inacessível durante a varredura: %v"
	DetPeloNome              = "Pelo nome %q, resolvido para %s."
	DetCaminhoEErroCitado    = "%q: %v"
	DetNaoEhDiretorio        = "%q existe mas não é um diretório."
	DetGrafiaVizinha         = "\n     Existe(m) ao lado, com grafia diferente: %s"
	DetNaoListou             = "Não foi possível listar %q: %v"
	DetEntradasRaiz          = "%d entradas na raiz."
	DetNaoEscreveu           = "Não foi possível escrever em %q: %v"
	DetObsidianAusente       = "Pasta `.obsidian` ausente: configurações, temas e plugins do Obsidian não serão detectados."
	DetNaoVerificou          = "Não foi possível verificar %q: %v"
	DetSemNotas              = "Nenhuma nota encontrada."
	DetNotas                 = "%d notas."
	DetCaminhoLongo          = "%d caracteres. Acima do limiar de %d: %s"
	DetMaiorCaminho          = "Maior caminho: %d caracteres."
	DetSemCacheDir           = "Nenhum diretório de cache configurado."
	DetNaoCriouCache         = "Não foi possível criar %q: %v"
	DetEspacoLivre           = "%d MB livres."
	DetNaoMediuEspaco        = "Não foi possível medir espaço livre: %v"

	DetLongPaths    = "`LongPathsEnabled` != 1 no registro e há caminho de %d caracteres: %s"
	DetSomenteNuvem = "%d nota(s) ainda não baixada(s) pelo sincronizador de nuvem."
	DetColisoes     = "%d colisão(ões): %s"

	DetNaoDerivou          = "Não foi possível derivar: %v"
	DetSondaFalhou         = "%v -- A ponte deste contexto vai servir em processo em vez de usar o *daemon*."
	DetSondaOK             = "Vale neste processo. Num host, o servidor roda noutro contexto, e a prova lá é a linha `conectado ao daemon` no log dele."
	DetHandshakeOK         = "*Handshake* completo."
	DetSemDaemon           = "Nenhum *daemon* rodando (a ponte servirá em processo)."
	DetHandshakeFalhou     = "Arquivo existe mas o *handshake* falhou: %v"
	DetCaminhoIndisponivel = "(caminho indisponível)"
	DetLogAusente          = "Ainda não existe (nenhum *daemon* rodou para este cofre)."
	DetLogResumo           = "%s (%d bytes, última escrita há %s)."
	DetCaminhoEErro        = "%s: %v"
	DetRuntimeAusente      = "Diretório de *runtime* ainda não existe."
	DetSemTravas           = "Nenhuma trava em uso."
	DetTravasEmUso         = "%d em %s: %s"
	DetTravaIlegivel       = "%s (não foi possível consultar: %v)"
	DetTravaComPID         = "%s (PID %d)"
	DetSocketEClasse       = "%s -- %s."
)

// O que existe no caminho do socket. O texto é o diagnóstico: "ausente" e
// "socket" são os dois estados saudáveis, e o resto é resíduo que impede o
// daemon de subir.
const (
	ClasseAusente     = "ausente"
	ClasseSocket      = "*socket*"
	ClasseDiretorio   = "DIRETÓRIO (nenhum *daemon* consegue usar este caminho)"
	ClasseSymlink     = "*symlink*"
	ClasseArquivo     = "arquivo comum de %d bytes (resíduo, nenhum *daemon* escuta aqui)"
	ClasseOutro       = "outro (modo=%v)"
	ClasseInacessivel = "inacessível (%v)"
)

// Os passos que a instalação anuncia antes de cada etapa começar, e o aviso de
// PATH que ela imprime no fim. Ver instalar.Sistema.Passo: quem anuncia é o
// pacote, e o texto é o que o usuário lê esperando.
const (
	PassoTrava     = "Tomando a trava de instalação."
	PassoProcessos = "Procurando processos em execução."
	PassoLimpeza   = "Limpando lixo de execuções anteriores."
	PassoChaves    = "Conferindo as chaves de cache."
	PassoBinario   = "Instalando o binário."
	PassoPath      = "Ajustando o *PATH*."
	PassoHosts     = "Configurando os hosts de IA."
	PassoManifesto = "Gravando o manifesto."

	AvisoPathWindows = "Abra um terminal NOVO para carregar o *PATH* atualizado."
	AvisoPathUnix    = "Abra um terminal NOVO, ou rode `source ~/.profile` para atualizar o *PATH*."
)

// Os hosts: o nome como aparece na tela, e o que cada um pede depois de ser
// configurado -- este sai no resumo da instalação, uma linha por host.
const (
	HostNomeClaudeDesktop  = "Claude Desktop"
	HostNomeClaudeCode     = "Claude Code (CLI)"
	HostNomeGeminiCLI      = "Gemini CLI"
	HostNomeAntigravity    = "Antigravity"
	HostNomeAntigravityIDE = "Antigravity IDE"
	HostNomeCodex          = "Codex CLI"
	HostNomeVSCode         = "VS Code"
	HostNomeCursor         = "Cursor"
	HostNomeWindsurf       = "Windsurf"

	HostClaudeDesktop  = "Reinicie o Claude Desktop para carregar o servidor."
	HostClaudeCode     = "Registrado no Claude Code."
	HostGeminiCLI      = "Registrado no Gemini CLI."
	HostAntigravity    = "Reinicie o Antigravity."
	HostAntigravityIDE = "Reinicie o Antigravity IDE."
	HostCodex          = "Registrado em `~/.codex/config.toml`."
	HostVSCode         = "Registrado na configuração de usuário do VS Code."
	HostCursor         = "Reinicie o Cursor."
	HostWindsurf       = "Reinicie o Windsurf."
)

// Erros do `daemon` que chegam ao terminal de quem o roda à mão. O log do
// daemon é outra coisa: sai por slog e não passa por aqui.
const (
	ErroLogCaminho   = "Resolvendo caminho do log do *daemon*: %w"
	ErroLogDiretorio = "Criando diretório do log do *daemon*: %w"
	ErroLogAbrir     = "Abrindo log do *daemon* %s: %w"
	ErroSocketDaemon = "Abrindo *socket* do *daemon*: %w"
)

// Como carregar o autocompletar em cada shell que o cobra não cobre sozinho.
// Saem SEM desenhar marcação: o que vem depois dos dois-pontos é para copiar, e
// a crase do tcsh faz parte do comando.
const (
	ShellNushell = "Acrescente ao seu config.nu:\n\n" +
		"  let gobsidian_completer = {|spans| gobsidian _carapace nushell ...$spans | from json }\n" +
		"  $env.config.completions.external = { enable: true, completer: $gobsidian_completer }"
	ShellElvish  = "Acrescente ao seu rc.elv:\n\n  eval (gobsidian completion elvish | slurp)"
	ShellIon     = "Acrescente ao seu initrc:\n\n  eval $(gobsidian completion ion)"
	ShellOil     = "Acrescente ao seu oshrc:\n\n  source <(gobsidian completion oil)"
	ShellTcsh    = "Acrescente ao seu .tcshrc:\n\n  eval `gobsidian completion tcsh`"
	ShellXonsh   = "Acrescente ao seu .xonshrc:\n\n  exec($(gobsidian completion xonsh))"
	ShellClink   = "Salve a saída em um arquivo .lua dentro do diretório de scripts do clink."
	ShellBashBLE = "Acrescente ao seu .bashrc, DEPOIS de carregar o ble.sh:\n\n  source <(gobsidian completion bash_ble)"
)

// Autocompletar: o que o shell mostra ao lado de cada valor. Sem marcação: o
// shell não a desenha.
const (
	CompletarNenhumHost  = "Não configura nenhum host."
	CompletarCofre       = "Cofre do Obsidian."
	CompletarCofreAberto = "Aberto agora."
	CompletarLogDebug    = "Tudo, inclusive o que só interessa depurando."
	CompletarLogInfo     = "O padrão."
	CompletarLogWarn     = "Só o que pede atenção."
	CompletarLogError    = "Só falha."
	CompletarResumoShell = "Gera o script de autocompletar para %s."
	CompletarDescricao   = "Gera o script de autocompletar para %s.\n\n%s"
)
