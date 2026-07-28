---
# YAML invalido de proposito: colchete de lista nunca fechado. Pina a
# garantia de que frontmatter quebrado nao derruba o parse — o corpo
# abaixo continua tendo heading, e FrontmatterErr registra o motivo em
# vez de Parse devolver erro.
titulo: [Direito Civil
tags: [civil
---
# Corpo apos frontmatter quebrado

Este texto ainda precisa aparecer nos headings.
