<!-- Exemplo citado no comentario de wikilinkParser.Parse: um terceiro colchete
faz o parser recusar "[[[a]]" em vez de chutar onde o wikilink comeca. Sem a
recusa, o link Markdown para d.md desapareceria — um link real perdido sob
qualquer leitura. Este fixture prova que o link markdown sobrevive. -->
[[[a]] b](d.md)
