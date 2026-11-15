package cog

// LimiteTabela permite aos testes forçar a leitura das tabelas de blocos sob demanda.
func LimiteTabela(n uint64) (restaurar func()) {
	antigo := limiteTabela
	limiteTabela = n
	return func() { limiteTabela = antigo }
}
