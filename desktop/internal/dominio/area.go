// Package dominio contém regras de negócio puras da Central (sem banco, sem UI).
// Regras: docs/requirements/regras-de-negocio.md (RN-40..45) e ADR-0009.
package dominio

import (
	"errors"
	"fmt"
	"strings"
)

// UnidadeArea é a unidade escolhida pelo usuário.
type UnidadeArea string

const (
	UnidadeHectare     UnidadeArea = "ha"
	UnidadeAlqPaulista UnidadeArea = "alq_paulista"
)

// dm² por milésimo da unidade (1 ha = 1.000.000 dm²; 1 alq. paulista = 2,42 ha).
var dm2PorMilesimo = map[UnidadeArea]int64{
	UnidadeHectare:     1000,
	UnidadeAlqPaulista: 2420,
}

// Erros de validação de área (códigos idênticos aos de shared/test-fixtures/area).
var (
	ErrAreaNegativa           = errors.New("negativo")
	ErrAreaMaisDe3Casas       = errors.New("mais_de_3_casas")
	ErrAreaNaoNumerica        = errors.New("nao_numerico")
	ErrAreaSeparadorMilhar    = errors.New("separador_de_milhar_nao_suportado")
	ErrAreaUnidadeObrigatoria = errors.New("unidade_obrigatoria")
	ErrAreaUnidadeInvalida    = errors.New("unidade_invalida")
)

// maxMilesimos limita a entrada para que somas em int64 nunca transbordem.
const maxMilesimos = int64(1_000_000_000_000) // 1 bilhão de unidades

// Area é um valor de área informado, preservado como digitado (em milésimos).
type Area struct {
	Milesimos int64
	Unidade   UnidadeArea
}

// UnidadeValida informa se u é uma unidade reconhecida.
func UnidadeValida(u UnidadeArea) bool { _, ok := dm2PorMilesimo[u]; return ok }

// ParseArea interpreta o texto digitado. Retorna (nil, nil) quando não há área
// (texto vazio). Aceita vírgula ou ponto como separador decimal, até 3 casas.
func ParseArea(entrada string, unidade *UnidadeArea) (*Area, error) {
	s := strings.TrimSpace(entrada)
	if s == "" {
		return nil, nil
	}
	if strings.HasPrefix(s, "-") {
		return nil, ErrAreaNegativa
	}
	if strings.Contains(s, ".") && strings.Contains(s, ",") {
		return nil, ErrAreaSeparadorMilhar
	}
	s = strings.ReplaceAll(s, ",", ".")
	inteira, frac, temFrac := strings.Cut(s, ".")
	if inteira == "" || !soDigitos(inteira) || (temFrac && (frac == "" || !soDigitos(frac))) {
		return nil, ErrAreaNaoNumerica
	}
	if len(frac) > 3 {
		return nil, ErrAreaMaisDe3Casas
	}
	frac += strings.Repeat("0", 3-len(frac))
	inteira = strings.TrimLeft(inteira, "0")
	if len(inteira) > 12 {
		return nil, fmt.Errorf("%w: valor grande demais", ErrAreaNaoNumerica)
	}
	var m int64
	for _, c := range inteira + frac {
		m = m*10 + int64(c-'0')
	}
	if m > maxMilesimos {
		return nil, fmt.Errorf("%w: valor grande demais", ErrAreaNaoNumerica)
	}
	if unidade == nil {
		return nil, ErrAreaUnidadeObrigatoria
	}
	if !UnidadeValida(*unidade) {
		return nil, ErrAreaUnidadeInvalida
	}
	return &Area{Milesimos: m, Unidade: *unidade}, nil
}

func soDigitos(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// Dm2 converte para decímetros quadrados inteiros (cálculo canônico e exato).
func (a Area) Dm2() int64 { return a.Milesimos * dm2PorMilesimo[a.Unidade] }

// FormatarHa formata dm² como hectares com 2 casas (HALF_UP) e vírgula decimal.
func FormatarHa(dm2 int64) string {
	centiHa := (dm2 + 5000) / 10000
	return fmt.Sprintf("%d,%02d", centiHa/100, centiHa%100)
}

// SomaAreas soma as áreas presentes; propriedades sem área (nil) não valem zero
// e são contadas à parte (RN-45).
func SomaAreas(areas []*Area) (totalDm2 int64, comArea, semArea int) {
	for _, a := range areas {
		if a == nil {
			semArea++
			continue
		}
		totalDm2 += a.Dm2()
		comArea++
	}
	return
}
