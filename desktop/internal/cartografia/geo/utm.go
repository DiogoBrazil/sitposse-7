package geo

import (
	"fmt"
	"math"
)

// Transverse Mercator (UTM) sobre o elipsoide WGS 84 pelas séries de Krüger de 6ª ordem
// (Karney, "Transverse Mercator with an accuracy of a few nanometers", J. Geodesy 2011).
// Erro < 5 nm dentro de 3.900 km do meridiano central: muito além do necessário para
// posicionar pixels de 2 m. Usado para ler imagens em UTM (EPSG:326xx/327xx; ADR-0027).

const (
	wgs84A = 6378137.0
	wgs84F = 1 / 298.257223563
	utmK0  = 0.9996
)

var tm = func() (r struct {
	e, a     float64
	alfa, bt [7]float64
}) {
	f := wgs84F
	n := f / (2 - f)
	n2, n3, n4, n5, n6 := n*n, n*n*n, n*n*n*n, n*n*n*n*n, n*n*n*n*n*n
	r.e = math.Sqrt(f * (2 - f))
	r.a = wgs84A / (1 + n) * (1 + n2/4 + n4/64 + n6/256)
	r.alfa = [7]float64{0,
		n/2 - 2*n2/3 + 5*n3/16 + 41*n4/180 - 127*n5/288 + 7891*n6/37800,
		13*n2/48 - 3*n3/5 + 557*n4/1440 + 281*n5/630 - 1983433*n6/1935360,
		61*n3/240 - 103*n4/140 + 15061*n5/26880 + 167603*n6/181440,
		49561*n4/161280 - 179*n5/168 + 6601661*n6/7257600,
		34729*n5/80640 - 3418889*n6/1995840,
		212378941 * n6 / 319334400,
	}
	r.bt = [7]float64{0,
		n/2 - 2*n2/3 + 37*n3/96 - n4/360 - 81*n5/512 + 96199*n6/604800,
		n2/48 + n3/15 - 437*n4/1440 + 46*n5/105 - 1118711*n6/3870720,
		17*n3/480 - 37*n4/840 - 209*n5/4480 + 5569*n6/90720,
		4397*n4/161280 - 11*n5/504 - 830251*n6/7257600,
		4583*n5/161280 - 108847*n6/3991680,
		20648693 * n6 / 638668800,
	}
	return
}()

// Zona UTM identificada pelo código EPSG (326zz norte, 327zz sul).
type Zona struct {
	Numero int
	Sul    bool
}

// ZonaEPSG devolve a zona de um código EPSG UTM/WGS 84.
func ZonaEPSG(epsg int) (Zona, bool) {
	switch {
	case epsg >= 32601 && epsg <= 32660:
		return Zona{epsg - 32600, false}, true
	case epsg >= 32701 && epsg <= 32760:
		return Zona{epsg - 32700, true}, true
	}
	return Zona{}, false
}

func (z Zona) meridiano() float64 { return float64(z.Numero*6-183) * math.Pi / 180 }

// UTM projeta latitude/longitude (graus, WGS 84) para leste/norte (metros) na zona dada,
// mesmo fora dela (as séries continuam válidas a milhares de quilômetros).
func UTM(p Ponto, z Zona) (leste, norte float64) {
	phi := p.Lat * math.Pi / 180
	lam := p.Lon*math.Pi/180 - z.meridiano()
	e := tm.e
	s := math.Sin(phi)
	t := math.Sinh(math.Atanh(s) - e*math.Atanh(e*s))
	xi1 := math.Atan2(t, math.Cos(lam))
	eta1 := math.Atanh(math.Sin(lam) / math.Sqrt(1+t*t))
	xi, eta := xi1, eta1
	for j := 1; j <= 6; j++ {
		k := float64(2 * j)
		xi += tm.alfa[j] * math.Sin(k*xi1) * math.Cosh(k*eta1)
		eta += tm.alfa[j] * math.Cos(k*xi1) * math.Sinh(k*eta1)
	}
	leste = 500000 + utmK0*tm.a*eta
	norte = utmK0 * tm.a * xi
	if z.Sul {
		norte += 10000000
	}
	return
}

// DeUTM é a inversa de UTM.
func DeUTM(leste, norte float64, z Zona) Ponto {
	if z.Sul {
		norte -= 10000000
	}
	xi := norte / (utmK0 * tm.a)
	eta := (leste - 500000) / (utmK0 * tm.a)
	xi1, eta1 := xi, eta
	for j := 1; j <= 6; j++ {
		k := float64(2 * j)
		xi1 -= tm.bt[j] * math.Sin(k*xi) * math.Cosh(k*eta)
		eta1 -= tm.bt[j] * math.Cos(k*xi) * math.Sinh(k*eta)
	}
	sh := math.Sinh(eta1)
	cx := math.Cos(xi1)
	tau1 := math.Sin(xi1) / math.Sqrt(sh*sh+cx*cx)
	lam := math.Atan2(sh, cx)
	// τ a partir de τ' (Newton; converge em 2–3 passos).
	e := tm.e
	e2 := e * e
	tau := tau1
	for i := 0; i < 8; i++ {
		sigma := math.Sinh(e * math.Atanh(e*tau/math.Sqrt(1+tau*tau)))
		taui := tau*math.Sqrt(1+sigma*sigma) - sigma*math.Sqrt(1+tau*tau)
		d := (tau1 - taui) / math.Sqrt(1+taui*taui) * (1 + (1-e2)*tau*tau) / ((1 - e2) * math.Sqrt(1+tau*tau))
		tau += d
		if math.Abs(d) < 1e-14 {
			break
		}
	}
	return Ponto{Lat: math.Atan(tau) * 180 / math.Pi, Lon: (lam + z.meridiano()) * 180 / math.Pi}
}

// ParaSRC converte lat/lon para as coordenadas do SRC (EPSG:4326 → lon/lat; UTM → leste/norte;
// EPSG:3857 → x/y Web Mercator).
func ParaSRC(epsg int, p Ponto) (x, y float64, err error) {
	switch {
	case epsg == 4326:
		return p.Lon, p.Lat, nil
	case epsg == 3857:
		x, y = XY(p)
		return x, y, nil
	}
	if z, ok := ZonaEPSG(epsg); ok {
		x, y = UTM(p, z)
		return x, y, nil
	}
	return 0, 0, fmt.Errorf("sistema de referência EPSG:%d não suportado", epsg)
}

// DeSRC é a inversa de ParaSRC.
func DeSRC(epsg int, x, y float64) (Ponto, error) {
	switch {
	case epsg == 4326:
		return Ponto{Lat: y, Lon: x}, nil
	case epsg == 3857:
		return LatLon(x, y), nil
	}
	if z, ok := ZonaEPSG(epsg); ok {
		return DeUTM(x, y, z), nil
	}
	return Ponto{}, fmt.Errorf("sistema de referência EPSG:%d não suportado", epsg)
}

// SuportaSRC diz se o EPSG pode ser lido.
func SuportaSRC(epsg int) bool {
	_, ok := ZonaEPSG(epsg)
	return ok || epsg == 4326 || epsg == 3857
}
