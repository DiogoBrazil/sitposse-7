package cartografia

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Armazem guarda, na pasta de dados, as imagens já consultadas e as cartas geradas,
// sem validade (ADR-0022). Só a área que o operador consultou é guardada; nada é baixado
// antecipadamente. "Limpar imagens" apaga só as imagens brutas, nunca as cartas.
type Armazem struct{ Dir string }

func (a Armazem) dirImagens() string { return filepath.Join(a.Dir, "imagens") }
func (a Armazem) dirCartas() string  { return filepath.Join(a.Dir, "cartas") }

// ChaveImagem identifica uma imagem pelos parâmetros que a determinam.
func ChaveImagem(provedor, versaoScript string, p PedidoImagem) string {
	partes := []string{provedor, versaoScript,
		fmt.Sprintf("%.2f,%.2f,%.2f,%.2f", p.Ret.MinX, p.Ret.MinY, p.Ret.MaxX, p.Ret.MaxY),
		fmt.Sprintf("%dx%d", p.Largura, p.Altura)}
	if p.Data != nil {
		partes = append(partes, "dia="+p.Data.UTC().Format("2006-01-02"))
	} else {
		partes = append(partes, "de="+p.De.UTC().Format(time.RFC3339), "ate="+p.Ate.UTC().Format(time.RFC3339))
	}
	if p.MaxNuvens != nil {
		partes = append(partes, fmt.Sprintf("nuvens<=%.1f", *p.MaxNuvens))
	}
	h := sha256.Sum256([]byte(strings.Join(partes, "|")))
	return hex.EncodeToString(h[:16])
}

var reChave = regexp.MustCompile(`^[0-9a-f]{32}$`)
var reCarta = regexp.MustCompile(`^[0-9]{8}-[0-9]{6}-[0-9a-f]{8}$`)

func gravarAtomico(destino string, dados []byte) error {
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(destino), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(dados); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), destino)
}

// GravarImagem guarda o PNG e os metadados (o PNG primeiro: metadado sem arquivo nunca existe).
func (a Armazem) GravarImagem(chave string, img *Imagem) error {
	if !reChave.MatchString(chave) {
		return errors.New("chave de imagem inválida")
	}
	if err := gravarAtomico(filepath.Join(a.dirImagens(), chave+".png"), img.PNG); err != nil {
		return err
	}
	meta, _ := json.Marshal(img)
	return gravarAtomico(filepath.Join(a.dirImagens(), chave+".json"), meta)
}

// LerImagem devolve a imagem guardada (ok=false se não houver).
func (a Armazem) LerImagem(chave string) (*Imagem, bool) {
	if !reChave.MatchString(chave) {
		return nil, false
	}
	meta, err := os.ReadFile(filepath.Join(a.dirImagens(), chave+".json"))
	if err != nil {
		return nil, false
	}
	pngBytes, err := os.ReadFile(filepath.Join(a.dirImagens(), chave+".png"))
	if err != nil {
		return nil, false
	}
	var img Imagem
	if json.Unmarshal(meta, &img) != nil {
		return nil, false
	}
	img.PNG = pngBytes
	return &img, true
}

// CaminhoImagem do PNG guardado (para a prévia), validando a chave.
func (a Armazem) CaminhoImagem(chave string) (string, bool) {
	if !reChave.MatchString(chave) {
		return "", false
	}
	p := filepath.Join(a.dirImagens(), chave+".png")
	_, err := os.Stat(p)
	return p, err == nil
}

// Uso do disco.
type Uso struct {
	Imagens      int   `json:"imagens"`
	BytesImagens int64 `json:"bytes_imagens"`
	Cartas       int   `json:"cartas"`
	BytesCartas  int64 `json:"bytes_cartas"`
}

func tamanhoDir(dir string, contar func(nome string) bool) (n int, total int64) {
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, e := d.Info(); e == nil {
			total += info.Size()
			if contar(d.Name()) {
				n++
			}
		}
		return nil
	})
	return
}

func (a Armazem) Uso() Uso {
	var u Uso
	u.Imagens, u.BytesImagens = tamanhoDir(a.dirImagens(), func(n string) bool { return strings.HasSuffix(n, ".png") })
	u.Cartas, u.BytesCartas = tamanhoDir(a.dirCartas(), func(n string) bool { return n == "carta.json" })
	return u
}

// LimparImagens apaga só as imagens brutas guardadas; as cartas ficam.
func (a Armazem) LimparImagens() (int, error) {
	es, err := os.ReadDir(a.dirImagens())
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range es {
		nome := e.Name()
		if e.IsDir() || !(strings.HasSuffix(nome, ".png") || strings.HasSuffix(nome, ".json")) {
			continue
		}
		if err := os.Remove(filepath.Join(a.dirImagens(), nome)); err != nil {
			return n, err
		}
		if strings.HasSuffix(nome, ".png") {
			n++
		}
	}
	return n, nil
}

// CartaGuardada: metadados de uma carta gerada.
type CartaGuardada struct {
	ID           string    `json:"id"`
	Titulo       string    `json:"titulo"`
	Tipo         string    `json:"tipo"` // individual | geral
	Propriedades int       `json:"propriedades"`
	Periodo      string    `json:"periodo"`
	Criterios    []string  `json:"criterios"`
	GeradaEm     time.Time `json:"gerada_em"`
	Arquivos     []string  `json:"arquivos"` // carta.pdf, carta.png
}

// GravarCarta guarda os arquivos (antes dos metadados) e devolve a carta registrada.
func (a Armazem) GravarCarta(c CartaGuardada, arquivos map[string][]byte) (CartaGuardada, error) {
	var b [4]byte
	_, _ = rand.Read(b[:])
	c.ID = c.GeradaEm.Local().Format("20060102-150405") + "-" + hex.EncodeToString(b[:])
	dir := filepath.Join(a.dirCartas(), c.ID)
	c.Arquivos = nil
	for nome, dados := range arquivos {
		if err := gravarAtomico(filepath.Join(dir, nome), dados); err != nil {
			return c, err
		}
		c.Arquivos = append(c.Arquivos, nome)
	}
	sort.Strings(c.Arquivos)
	meta, _ := json.MarshalIndent(c, "", " ")
	return c, gravarAtomico(filepath.Join(dir, "carta.json"), meta)
}

// ListarCartas, da mais recente para a mais antiga. Funciona sem internet.
func (a Armazem) ListarCartas() []CartaGuardada {
	es, _ := os.ReadDir(a.dirCartas())
	out := []CartaGuardada{}
	for _, e := range es {
		if !e.IsDir() || !reCarta.MatchString(e.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(a.dirCartas(), e.Name(), "carta.json"))
		if err != nil {
			continue // carta interrompida antes dos metadados: ignorada
		}
		var c CartaGuardada
		if json.Unmarshal(b, &c) == nil {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GeradaEm.After(out[j].GeradaEm) })
	return out
}

// CaminhoCarta de um arquivo de uma carta guardada (id e nome validados).
func (a Armazem) CaminhoCarta(id, arquivo string) (string, error) {
	if !reCarta.MatchString(id) || (arquivo != "carta.pdf" && arquivo != "carta.png") {
		return "", errors.New("carta inexistente")
	}
	p := filepath.Join(a.dirCartas(), id, arquivo)
	if _, err := os.Stat(p); err != nil {
		return "", errors.New("carta inexistente")
	}
	return p, nil
}
