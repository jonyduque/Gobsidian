<!-- Pina o comportamento documentado em wikilinkParser.Parse para colchete
triplo: o parser recusa a primeira oferta do gatilho (offset+2 acha '['), e o
goldmark reoferece o gatilho um byte adiante — onde a analise deixa de ser
ambigua. O resultado exato (link para "triplo" ou nenhum link) e questao de
paridade em aberto para a Task 25; este golden fixa o que o codigo de hoje
produz para que uma mudanca futura seja uma decisao visivel, nao um acidente. -->
[[[triplo]]]
