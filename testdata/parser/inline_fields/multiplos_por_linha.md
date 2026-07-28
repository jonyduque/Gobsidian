<!-- Duas formas do Dataview lado a lado: sem colchetes, so um campo por
linha vale (o valor de "a" absorve "1 b:: 2" inteiro, regra 2 barra o
segundo "::" de virar campo novo porque "1 b" nao comeca a linha). Entre
colchetes, cada campo tem seu proprio terminador ']' e os dois contam.
Como as duas formas usam a mesma chave "a", Inline["a"] acumula as DUAS
ocorrencias na ordem do documento — ["1 b:: 2", "1"] — a mesma regra de
chave repetida que TestInlineFieldRepeatedKey cobre, so que entre formas
diferentes em vez da mesma forma duas vezes. -->
a:: 1 b:: 2

[a:: 1] [b:: 2]
