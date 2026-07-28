<!-- Tres formas rejeitadas, cada uma em seu proprio paragrafo para nao
interagir: marcador no MEIO da linha (nao no fim), marcador que nao esta na
ULTIMA linha do paragrafo, e um "^id" seguido de caractere fora do alfabeto
(espaco seguido de mais texto, nao so espaco em branco ate o fim da linha).
As tres devem produzir zero blocos. -->
Marcador no meio da linha ^abc123 mais texto depois.

Primeira linha nao conta ^nope
Segunda linha e a ultima, sem marcador nenhum.

Caracteres invalidos depois do id ^abc def
