// Package cartografia reúne o provedor de imagens de satélite, o armazenamento local das
// imagens e cartas e a montagem das cartas-imagem (A27, ADR-0022; ADR-0013).
//
// Este é o ÚNICO ponto da Central que fala com a internet (desktop/AGENTS.md). Ao provedor
// vão só o retângulo geográfico, o período e os parâmetros da imagem: nunca dados de cadastro.
package cartografia

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sitposse7/desktop/internal/cartografia/geo"
)

// Erros com mensagem para o operador. O app mostra a mensagem amigável da RN-91
// quando o erro é ErrSemConexao.
var (
	ErrSemCredenciais       = errors.New("configure as credenciais do Copernicus em Cartas-imagem → Configurar")
	ErrCredenciaisInvalidas = errors.New("o Copernicus recusou as credenciais (confira o identificador e o segredo do cliente OAuth)")
	ErrSemConexao           = errors.New("O mapa e a carta-imagem precisam de conexão com a internet. Seus cadastros, coordenadas, estatísticas e filtros continuam disponíveis offline. Conecte-se e tente novamente.")
	ErrResposta             = errors.New("o Copernicus devolveu uma resposta inesperada")
	ErrImagemIncompleta     = errors.New("a imagem desta data não cobre toda a área (há partes sem dados); escolha outra data ou o mosaico do período")
)

// ErrCota: limite de requisições ou de unidades de processamento atingido.
type ErrCota struct{ TentarApos time.Duration }

func (e *ErrCota) Error() string {
	if e.TentarApos > 0 {
		return fmt.Sprintf("limite de uso do Copernicus atingido; tente de novo em %s (a cota gratuita é mensal e por minuto)", e.TentarApos.Round(time.Second))
	}
	return "limite de uso do Copernicus atingido (a cota gratuita é renovada no dia 1º de cada mês)"
}

// Passagem é uma aquisição disponível sobre a área (um dia de passagem do satélite).
type Passagem struct {
	Data   time.Time `json:"data"`   // data (UTC) da aquisição
	Nuvens float64   `json:"nuvens"` // cobertura de nuvens estimada (%), a menor entre os tiles do dia
	Tiles  int       `json:"tiles"`  // quantos tiles do catálogo cobrem a área nesse dia
}

// PedidoImagem descreve a imagem desejada. Data != nil pede uma passagem única
// (a carta cita a data); senão é um mosaico do período De–Ate (a carta cita o período).
type PedidoImagem struct {
	Ret       geo.Retangulo
	Largura   int
	Altura    int
	Data      *time.Time
	De, Ate   time.Time
	MaxNuvens *float64
}

// Imagem obtida (PNG RGBA, com transparência onde não há dados).
type Imagem struct {
	PNG              []byte        `json:"-"`
	Ret              geo.Retangulo `json:"ret"`
	Largura          int           `json:"largura"`
	Altura           int           `json:"altura"`
	Data             *time.Time    `json:"data"`
	De               time.Time     `json:"de"`
	Ate              time.Time     `json:"ate"`
	SemDados         float64       `json:"sem_dados"` // fração de pixels sem dados (0..1)
	Fonte            string        `json:"fonte"`
	Colecao          string        `json:"colecao"`
	ResolucaoNativaM float64       `json:"resolucao_nativa_m"`
	ObtidaEm         time.Time     `json:"obtida_em"`
}

// Periodo descreve a data ou o período da imagem para a carta (nunca uma data única num mosaico).
func (i Imagem) Periodo() string {
	if i.Data != nil {
		return "Aquisição de " + i.Data.Format("02/01/2006")
	}
	return fmt.Sprintf("Mosaico de aquisições entre %s e %s (menor nebulosidade)", i.De.Format("02/01/2006"), i.Ate.Format("02/01/2006"))
}

// Provedor de imagens de satélite (ADR-0013): o CDSE é a primeira implementação.
type Provedor interface {
	Nome() string
	// Atribuicao exigida pela licença dos dados, para o ano da imagem.
	Atribuicao(ano int) string
	Passagens(ctx context.Context, ret geo.Retangulo, de, ate time.Time) ([]Passagem, error)
	Imagem(ctx context.Context, p PedidoImagem) (*Imagem, error)
}
