package cartografia

import (
	"fmt"
	"time"
)

// Fontes de imagem (ADR-0027). A ordem é a preferência quando a qualidade empata.
const (
	FonteCBERS   = "cbers4a-wpm"   // CBERS-4A/WPM fusionado (2 m), INPE
	FonteS2      = "sentinel2-l2a" // Sentinel-2 L2A (10 m), distribuído pelo INPE
	FonteMosaico = "mosaico-s2"    // mosaico trimestral Sentinel-2 do INPE (10 m)
	FonteCDSE    = "cdse-s2"       // Sentinel-2 L2A pelo Copernicus (reserva)
)

// Candidata é uma cena (ou mosaico) que pode compor a imagem da carta.
type Candidata struct {
	ID         string     `json:"id"`
	Fonte      string     `json:"fonte"`
	Data       time.Time  `json:"data"`          // data de aquisição (cena) ou início (mosaico)
	Ate        *time.Time `json:"ate,omitempty"` // fim do período (mosaico)
	ResolucaoM float64    `json:"resolucao_m"`
	URLRGB     string     `json:"url_rgb"`
	URLSCL     string     `json:"url_scl,omitempty"` // classificação de cena (Sentinel-2)
	Nuvens     *float64   `json:"nuvens_catalogo,omitempty"`
	BBox       [4]float64 `json:"bbox"`
}

// Rotulo para o operador e para a carta.
func (c Candidata) Rotulo() string {
	switch c.Fonte {
	case FonteCBERS:
		return "CBERS-4A/WPM (INPE)"
	case FonteS2:
		return "Sentinel-2 L2A (Copernicus/INPE)"
	case FonteMosaico:
		return "Mosaico Sentinel-2 trimestral (INPE)"
	case FonteCDSE:
		return "Sentinel-2 L2A (Copernicus Data Space)"
	}
	return c.Fonte
}

// Periodo: "13/10/2026" ou "01/04/2026 a 30/06/2026".
func (c Candidata) Periodo() string {
	if c.Ate != nil {
		return fmt.Sprintf("%s a %s", c.Data.Format("02/01/2006"), c.Ate.Format("02/01/2006"))
	}
	return c.Data.Format("02/01/2006")
}

// Atribuicao exigida pela licença dos dados.
func (c Candidata) Atribuicao() string {
	ano := c.Data.Year()
	if c.Ate != nil {
		ano = c.Ate.Year()
	}
	switch c.Fonte {
	case FonteCBERS:
		return "Imagem CBERS-4A/WPM © INPE (CC BY 4.0)"
	case FonteMosaico:
		return fmt.Sprintf("Contains modified Copernicus Sentinel data %d; mosaico Brazil Data Cube/INPE", ano)
	default:
		return fmt.Sprintf("Contains modified Copernicus Sentinel data %d", ano)
	}
}

// PrioridadeFonte: menor = preferida no desempate.
func PrioridadeFonte(f string) int {
	switch f {
	case FonteCBERS:
		return 0
	case FonteS2:
		return 1
	case FonteCDSE:
		return 2
	}
	return 3
}
