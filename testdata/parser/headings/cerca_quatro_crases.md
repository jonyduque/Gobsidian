<!-- CommonMark exige que a cerca de fechamento tenha PELO MENOS o comprimento
da de abertura: uma linha de tres crases dentro de uma cerca de quatro NAO a
fecha. Sem essa regra "# Depois" seria lido como heading real, quando na
verdade ainda esta dentro do bloco de codigo — a hierarquia inteira sairia
errada numa nota que documenta Markdown com exemplos aninhados. -->
# Antes

````go
codigo
```
ainda dentro da cerca
````

# Depois
