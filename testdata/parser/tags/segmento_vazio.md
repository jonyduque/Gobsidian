<!-- Barra dupla interna produz um segmento vazio de hierarquia ao dividir por
'/' — collapseTagSlashes colapsa para "a/b", nao "a//b". Sem isso tag_list
com hierarchical:true criaria um no sem nome onde toda tag com segmento
vazio na mesma profundidade colide. -->
Veja #a//b depois.
