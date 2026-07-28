<!-- Fora de qualquer bloco de codigo: o escape do CommonMark consome a barra
invertida e o caractere seguinte como texto literal antes que wikilink, tag
ou block-id cheguem a ver o gatilho. Mesmo mecanismo que ja cobre
TestWikilinkSuppressedInCode/escapado, estendido para os tres tipos de
marcador de uma vez. -->
\[\[nao e link\]\] \#nao-e-tag texto \^nao-e-bloco
