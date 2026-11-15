// Package contratos define os payloads JSON v2 trocados com o app Campo
// (shared/schemas/*.v2.schema.json, ADR-0024) e a validação deles em Go.
//
// A Central não usa a biblioteca de JSON Schema em produção (só nos testes):
// os testes garantem que estas validações e os schemas concordam em todas as
// fixtures de shared/test-fixtures/contratos.
package contratos

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"sitposse7/desktop/internal/codigos"
)

// Versões aceitas por esta Central.
const (
	ProtocoloVersao = 2
	SchemaVersao    = 2
)

// Limites (iguais aos dos schemas).
const (
	MaxTexto        = 200
	MaxTextoLongo   = 4000 // observações relevantes (RN-15), em caracteres Unicode
	MaxQuantidade   = 1_000_000
	MaxTamanhoFoto  = 50 * 1024 * 1024
	MaxPropriedades = 5000
	MaxItens        = 50000
	MaxFotos        = 20000
)

const formatoInstante = "2006-01-02T15:04:05.000Z"

var (
	reUUID     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	reInstante = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)
	reSHA256   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	reCodigo   = regexp.MustCompile(`^(eletro|movel|ave)\.[a-z_]+$`)
)

// FormatarInstante converte epoch em ms (UTC) para ISO 8601 com ms e Z (ADR-0010).
func FormatarInstante(ms int64) string {
	return time.UnixMilli(ms).UTC().Format(formatoInstante)
}

// LerInstante converte ISO 8601 UTC com ms e Z para epoch em ms.
func LerInstante(s string) (int64, error) {
	if !reInstante.MatchString(s) {
		return 0, fmt.Errorf("instante fora do formato ISO 8601 UTC com ms: %q", s)
	}
	t, err := time.Parse(formatoInstante, s)
	if err != nil {
		return 0, fmt.Errorf("instante inválido %q: %w", s, err)
	}
	return t.UnixMilli(), nil
}

// UUIDValido confere UUID v4 em minúsculas.
func UUIDValido(s string) bool { return reUUID.MatchString(s) }

// Coleta é o conjunto de cadastros feitos no celular com um código de equipe (RN-05, ADR-0024).
type Coleta struct {
	UUID           string `json:"uuid"`
	CodigoEquipe   string `json:"codigo_equipe"`
	CodigoOriginal string `json:"codigo_original"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	Rev            int64  `json:"rev"`
}

// Propriedade é o cadastro de propriedade/família (RN-10..55).
type Propriedade struct {
	UUID                      string   `json:"uuid"`
	ColetaUUID                string   `json:"coleta_uuid"`
	NomePropriedade           string   `json:"nome_propriedade"`
	ResponsavelFamilia        string   `json:"responsavel_familia"`
	ObservacoesRelevantes     *string  `json:"observacoes_relevantes"`
	PessoasAdultos            int64    `json:"pessoas_adultos"`
	PessoasJovens             int64    `json:"pessoas_jovens"`
	PessoasCriancas           int64    `json:"pessoas_criancas"`
	VeicCaminhoes             int64    `json:"veic_caminhoes"`
	VeicAutomoveisUtilitarios int64    `json:"veic_automoveis_utilitarios"`
	VeicMotosCiclomotores     int64    `json:"veic_motos_ciclomotores"`
	BovAdultos                int64    `json:"bov_adultos"`
	BovBezerros               int64    `json:"bov_bezerros"`
	EquAdultos                int64    `json:"equ_adultos"`
	EquPotros                 int64    `json:"equ_potros"`
	AreaValorMilesimos        *int64   `json:"area_valor_milesimos"`
	AreaUnidade               *string  `json:"area_unidade"`
	Lat                       *float64 `json:"lat"`
	Lon                       *float64 `json:"lon"`
	GPSOrigem                 *string  `json:"gps_origem"`
	GPSPrecisaoM              *float64 `json:"gps_precisao_m"`
	GPSCapturadoEm            *string  `json:"gps_capturado_em"`
	GPSSatelites              *int64   `json:"gps_satelites"`
	CreatedAt                 string   `json:"created_at"`
	UpdatedAt                 string   `json:"updated_at"`
	Rev                       int64    `json:"rev"`
	ExcluidoEm                *string  `json:"excluido_em"`
}

// Item é um item dinâmico da propriedade (RN-30..35).
type Item struct {
	UUID              string  `json:"uuid"`
	PropriedadeUUID   string  `json:"propriedade_uuid"`
	Categoria         string  `json:"categoria"`
	CodigoCatalogo    string  `json:"codigo_catalogo"`
	NomePersonalizado *string `json:"nome_personalizado"`
	Quantidade        int64   `json:"quantidade"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	Rev               int64   `json:"rev"`
	ExcluidoEm        *string `json:"excluido_em"`
}

// Foto são os metadados de uma fotografia (RN-60..63). Os bytes vão por PUT /v1/fotos.
type Foto struct {
	UUID            string  `json:"uuid"`
	PropriedadeUUID string  `json:"propriedade_uuid"`
	Mime            string  `json:"mime"`
	TamanhoBytes    int64   `json:"tamanho_bytes"`
	SHA256          string  `json:"sha256"`
	Largura         int64   `json:"largura"`
	Altura          int64   `json:"altura"`
	CapturadaEm     string  `json:"capturada_em"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	Rev             int64   `json:"rev"`
	ExcluidoEm      *string `json:"excluido_em"`
}

// Lote é o corpo de PUT /v1/lotes/{lote_uuid}.
type Lote struct {
	ProtocoloVersao int           `json:"protocolo_versao"`
	SchemaVersao    int           `json:"schema_versao"`
	LoteUUID        string        `json:"lote_uuid"`
	InstalacaoUUID  string        `json:"instalacao_uuid"`
	CriadoEm        string        `json:"criado_em"`
	Coleta          Coleta        `json:"coleta"`
	Propriedades    []Propriedade `json:"propriedades"`
	Itens           []Item        `json:"itens"`
	Fotos           []Foto        `json:"fotos"`
}

// FotoFaltante indica quantos bytes de uma foto a Central já tem em staging.
type FotoFaltante struct {
	UUID   string `json:"uuid"`
	Offset int64  `json:"offset"`
}

// Estados do lote na Central (estado-lote.v2).
const (
	StatusRecebendo  = "recebendo"
	StatusAguardando = "aguardando_confirmacao"
	StatusImportado  = "importado"
	StatusCancelado  = "cancelado"
)

// Motivos de cancelamento.
const (
	MotivoCanceladoOperador = "cancelado_operador"
	MotivoCodigoInexistente = "codigo_inexistente"
)

// EstadoLote é a resposta de PUT/GET /v1/lotes/{lote} e do importar (estado-lote.v2).
type EstadoLote struct {
	ProtocoloVersao int            `json:"protocolo_versao"`
	LoteUUID        string         `json:"lote_uuid"`
	Status          string         `json:"status"`
	FotosFaltantes  []FotoFaltante `json:"fotos_faltantes"`
	Recibo          *Recibo        `json:"recibo"`
	Motivo          *string        `json:"motivo"`
}

// Resultados possíveis da aplicação de um registro (protocolo.md §6).
const (
	ResultadoAplicado  = "aplicado"
	ResultadoDuplicado = "duplicado"
	ResultadoConflito  = "conflito"
	ResultadoRejeitado = "rejeitado"
)

// ResultadoRecibo é uma linha do recibo.
type ResultadoRecibo struct {
	Entidade     string `json:"entidade"`
	UUID         string `json:"uuid"`
	Rev          int64  `json:"rev"`
	Resultado    string `json:"resultado"`
	Motivo       string `json:"motivo,omitempty"`
	ConflitoUUID string `json:"conflito_uuid,omitempty"`
}

// Destino é a equipe confirmada pelo operador para a coleta do lote (ADR-0024).
type Destino struct {
	EstudoCodigo    string `json:"estudo_codigo"`
	EstudoNome      string `json:"estudo_nome"`
	JornadaNumero   int64  `json:"jornada_numero"`
	JornadaData     string `json:"jornada_data"`
	EquipeNome      string `json:"equipe_nome"`
	EquipeCodigo    string `json:"equipe_codigo"`
	CodigoInformado string `json:"codigo_informado"`
	Corrigido       bool   `json:"corrigido"`
}

// Recibo é o comprovante persistido da importação de um lote (recibo.v2).
type Recibo struct {
	ProtocoloVersao int               `json:"protocolo_versao"`
	LoteUUID        string            `json:"lote_uuid"`
	CentralID       string            `json:"central_id"`
	ImportadoEm     string            `json:"importado_em"`
	Destino         Destino           `json:"destino"`
	Resultados      []ResultadoRecibo `json:"resultados"`
	ReciboSHA256    string            `json:"recibo_sha256"`
}

// TextoCanonico é a entrada do recibo_sha256 (descrição em recibo.v2.schema.json).
func (r *Recibo) TextoCanonico() string {
	var b strings.Builder
	b.WriteString(r.LoteUUID + "\n" + r.CentralID + "\n" + r.ImportadoEm + "\n")
	d := r.Destino
	b.WriteString("destino|" + d.EstudoCodigo + "|" + strconv.FormatInt(d.JornadaNumero, 10) + "|" + d.JornadaData + "|" +
		d.EquipeCodigo + "|" + d.CodigoInformado + "|" + strconv.FormatBool(d.Corrigido) + "\n")
	for _, x := range r.Resultados {
		b.WriteString(x.Entidade + "|" + x.UUID + "|" + strconv.FormatInt(x.Rev, 10) + "|" + x.Resultado + "|" + x.Motivo + "\n")
	}
	return b.String()
}

// CalcularHash devolve o recibo_sha256 do conteúdo atual.
func (r *Recibo) CalcularHash() string {
	h := sha256.Sum256([]byte(r.TextoCanonico()))
	return hex.EncodeToString(h[:])
}

// PedidoPareamento é o corpo de POST /v1/pareamento.
type PedidoPareamento struct {
	ProtocoloVersao int     `json:"protocolo_versao"`
	SchemaVersao    int     `json:"schema_versao"`
	Token           string  `json:"token_pareamento"`
	InstalacaoUUID  string  `json:"instalacao_uuid"`
	RotuloAparelho  *string `json:"rotulo_aparelho,omitempty"`
}

// RespostaPareamento é a resposta 200 de POST /v1/pareamento.
type RespostaPareamento struct {
	ProtocoloVersao       int    `json:"protocolo_versao"`
	CentralID             string `json:"central_id"`
	CredencialDispositivo string `json:"credencial_dispositivo"`
}

// ---------- validação ----------

func textoObrigatorio(campo, v string) error {
	if strings.TrimSpace(v) == "" {
		return fmt.Errorf("%s vazio", campo)
	}
	if utf8.RuneCountInString(v) > MaxTexto {
		return fmt.Errorf("%s com mais de %d caracteres", campo, MaxTexto)
	}
	return nil
}

func textoOpcional(campo string, v *string) error {
	if v != nil && utf8.RuneCountInString(*v) > MaxTexto {
		return fmt.Errorf("%s com mais de %d caracteres", campo, MaxTexto)
	}
	return nil
}

func textoLongoOpcional(campo string, v *string) error {
	if v == nil {
		return nil
	}
	if strings.TrimSpace(*v) == "" {
		return fmt.Errorf("%s vazio deve ir como null", campo)
	}
	if utf8.RuneCountInString(*v) > MaxTextoLongo {
		return fmt.Errorf("%s com mais de %d caracteres", campo, MaxTextoLongo)
	}
	return nil
}

func uuidCampo(campo, v string) error {
	if !UUIDValido(v) {
		return fmt.Errorf("%s não é UUID v4 em minúsculas: %q", campo, v)
	}
	return nil
}

func instanteCampo(campo, v string) error {
	if _, err := LerInstante(v); err != nil {
		return fmt.Errorf("%s: %w", campo, err)
	}
	return nil
}

func instanteOpcional(campo string, v *string) error {
	if v == nil {
		return nil
	}
	return instanteCampo(campo, *v)
}

func revCampo(v int64) error {
	if v < 1 {
		return fmt.Errorf("rev deve ser ≥ 1 (recebido %d)", v)
	}
	return nil
}

func primeiroErro(erros ...error) error {
	for _, e := range erros {
		if e != nil {
			return e
		}
	}
	return nil
}

func codigoEquipe(campo, v string) error {
	if !codigos.Valido(v, codigos.PrefixoEquipe) {
		return fmt.Errorf("%s inválido: %q", campo, v)
	}
	return nil
}

// ValidarColeta aplica RN-02 e RN-05.
func ValidarColeta(c *Coleta) error {
	return primeiroErro(
		uuidCampo("coleta.uuid", c.UUID),
		codigoEquipe("coleta.codigo_equipe", c.CodigoEquipe),
		codigoEquipe("coleta.codigo_original", c.CodigoOriginal),
		instanteCampo("coleta.created_at", c.CreatedAt),
		instanteCampo("coleta.updated_at", c.UpdatedAt),
		revCampo(c.Rev),
	)
}

// ValidarPropriedade aplica RN-11..55 (e RN-15) a um registro recebido.
func ValidarPropriedade(p *Propriedade) error {
	if err := primeiroErro(
		uuidCampo("uuid", p.UUID),
		uuidCampo("coleta_uuid", p.ColetaUUID),
		textoObrigatorio("nome da propriedade", p.NomePropriedade),
		textoObrigatorio("nome do responsável", p.ResponsavelFamilia),
		textoLongoOpcional("observações relevantes", p.ObservacoesRelevantes),
		instanteCampo("created_at", p.CreatedAt),
		instanteCampo("updated_at", p.UpdatedAt),
		instanteOpcional("excluido_em", p.ExcluidoEm),
		revCampo(p.Rev),
	); err != nil {
		return err
	}
	qs := map[string]int64{
		"pessoas_adultos": p.PessoasAdultos, "pessoas_jovens": p.PessoasJovens, "pessoas_criancas": p.PessoasCriancas,
		"veic_caminhoes": p.VeicCaminhoes, "veic_automoveis_utilitarios": p.VeicAutomoveisUtilitarios, "veic_motos_ciclomotores": p.VeicMotosCiclomotores,
		"bov_adultos": p.BovAdultos, "bov_bezerros": p.BovBezerros, "equ_adultos": p.EquAdultos, "equ_potros": p.EquPotros,
	}
	for campo, q := range qs {
		if q < 0 || q > MaxQuantidade {
			return fmt.Errorf("%s fora da faixa 0..%d: %d", campo, MaxQuantidade, q)
		}
	}
	// Área (RN-40..44): valor e unidade juntos.
	if (p.AreaValorMilesimos == nil) != (p.AreaUnidade == nil) {
		return fmt.Errorf("área e unidade devem vir juntas")
	}
	if p.AreaValorMilesimos != nil {
		if *p.AreaValorMilesimos < 0 {
			return fmt.Errorf("área negativa")
		}
		if *p.AreaUnidade != "ha" && *p.AreaUnidade != "alq_paulista" {
			return fmt.Errorf("unidade de área inválida: %q", *p.AreaUnidade)
		}
	}
	// Coordenadas (RN-50..53).
	if p.Lat == nil {
		if p.Lon != nil || p.GPSOrigem != nil || p.GPSPrecisaoM != nil || p.GPSCapturadoEm != nil || p.GPSSatelites != nil {
			return fmt.Errorf("dados de GPS sem latitude")
		}
		return nil
	}
	if p.Lon == nil {
		return fmt.Errorf("latitude sem longitude")
	}
	if *p.Lat < -90 || *p.Lat > 90 || *p.Lon < -180 || *p.Lon > 180 {
		return fmt.Errorf("coordenada fora da faixa: %v, %v", *p.Lat, *p.Lon)
	}
	if p.GPSOrigem == nil || (*p.GPSOrigem != "gnss" && *p.GPSOrigem != "manual") {
		return fmt.Errorf("origem do GPS deve ser gnss ou manual")
	}
	if *p.GPSOrigem == "manual" && p.GPSPrecisaoM != nil {
		return fmt.Errorf("coordenada manual não tem precisão (RN-52)")
	}
	if p.GPSPrecisaoM != nil && *p.GPSPrecisaoM < 0 {
		return fmt.Errorf("precisão negativa")
	}
	if p.GPSSatelites != nil && *p.GPSSatelites < 0 {
		return fmt.Errorf("satélites negativo")
	}
	return instanteOpcional("gps_capturado_em", p.GPSCapturadoEm)
}

// CategoriaDoCodigo devolve a categoria implícita no prefixo do código do catálogo.
func CategoriaDoCodigo(codigo string) string {
	switch {
	case strings.HasPrefix(codigo, "eletro."):
		return "eletrodomestico"
	case strings.HasPrefix(codigo, "movel."):
		return "movel"
	case strings.HasPrefix(codigo, "ave."):
		return "ave"
	}
	return ""
}

// EhOutros indica um código "Outros" (nome livre obrigatório, RN-33).
func EhOutros(codigo string) bool { return strings.HasSuffix(codigo, ".outros") }

// ValidarItem aplica RN-31..34 (o catálogo é conferido na importação).
func ValidarItem(i *Item) error {
	if err := primeiroErro(
		uuidCampo("uuid", i.UUID),
		uuidCampo("propriedade_uuid", i.PropriedadeUUID),
		instanteCampo("created_at", i.CreatedAt),
		instanteCampo("updated_at", i.UpdatedAt),
		instanteOpcional("excluido_em", i.ExcluidoEm),
		revCampo(i.Rev),
	); err != nil {
		return err
	}
	if !reCodigo.MatchString(i.CodigoCatalogo) {
		return fmt.Errorf("código de catálogo inválido: %q", i.CodigoCatalogo)
	}
	if i.Categoria != CategoriaDoCodigo(i.CodigoCatalogo) {
		return fmt.Errorf("categoria %q não corresponde ao código %q", i.Categoria, i.CodigoCatalogo)
	}
	if EhOutros(i.CodigoCatalogo) {
		if i.NomePersonalizado == nil {
			return fmt.Errorf("'Outros' sem nome")
		}
		if err := textoObrigatorio("nome do item 'Outros'", *i.NomePersonalizado); err != nil {
			return err
		}
	} else if i.NomePersonalizado != nil {
		return fmt.Errorf("item do catálogo não tem nome livre")
	}
	if i.Quantidade < 0 || i.Quantidade > MaxQuantidade {
		return fmt.Errorf("quantidade fora da faixa 0..%d: %d", MaxQuantidade, i.Quantidade)
	}
	return nil
}

// ValidarFoto aplica RN-60..62 aos metadados.
func ValidarFoto(f *Foto) error {
	if err := primeiroErro(
		uuidCampo("uuid", f.UUID),
		uuidCampo("propriedade_uuid", f.PropriedadeUUID),
		instanteCampo("capturada_em", f.CapturadaEm),
		instanteCampo("created_at", f.CreatedAt),
		instanteCampo("updated_at", f.UpdatedAt),
		instanteOpcional("excluido_em", f.ExcluidoEm),
		revCampo(f.Rev),
	); err != nil {
		return err
	}
	if f.Mime != "image/jpeg" {
		return fmt.Errorf("tipo de foto não suportado: %q", f.Mime)
	}
	if f.TamanhoBytes < 1 || f.TamanhoBytes > MaxTamanhoFoto {
		return fmt.Errorf("tamanho da foto fora da faixa: %d", f.TamanhoBytes)
	}
	if !reSHA256.MatchString(f.SHA256) {
		return fmt.Errorf("sha256 deve ter 64 dígitos hexadecimais minúsculos")
	}
	if f.Largura < 1 || f.Altura < 1 {
		return fmt.Errorf("dimensões inválidas: %dx%d", f.Largura, f.Altura)
	}
	return nil
}

// ValidarEnvelope confere o que torna o lote inteiro inaceitável (resposta 400/422).
// Registros individuais inválidos não invalidam o lote: viram "rejeitado" no recibo.
func ValidarEnvelope(l *Lote) error {
	if l.ProtocoloVersao != ProtocoloVersao || l.SchemaVersao != SchemaVersao {
		return fmt.Errorf("versão não suportada (protocolo %d, esquema %d); esta Central aceita protocolo %d e esquema %d",
			l.ProtocoloVersao, l.SchemaVersao, ProtocoloVersao, SchemaVersao)
	}
	if err := primeiroErro(
		uuidCampo("lote_uuid", l.LoteUUID),
		uuidCampo("instalacao_uuid", l.InstalacaoUUID),
		instanteCampo("criado_em", l.CriadoEm),
		ValidarColeta(&l.Coleta),
	); err != nil {
		return err
	}
	if len(l.Propriedades) > MaxPropriedades || len(l.Itens) > MaxItens || len(l.Fotos) > MaxFotos {
		return fmt.Errorf("lote grande demais")
	}
	return nil
}

// ValidarLoteCompleto valida envelope e registros (usado nos testes de contrato).
func ValidarLoteCompleto(l *Lote) error {
	if err := ValidarEnvelope(l); err != nil {
		return err
	}
	for i := range l.Propriedades {
		if err := ValidarPropriedade(&l.Propriedades[i]); err != nil {
			return fmt.Errorf("propriedade %d: %w", i, err)
		}
	}
	for i := range l.Itens {
		if err := ValidarItem(&l.Itens[i]); err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}
	}
	for i := range l.Fotos {
		if err := ValidarFoto(&l.Fotos[i]); err != nil {
			return fmt.Errorf("foto %d: %w", i, err)
		}
	}
	return nil
}

// ValidarPedidoPareamento confere o pedido de pareamento.
func ValidarPedidoPareamento(p *PedidoPareamento) error {
	if p.ProtocoloVersao != ProtocoloVersao || p.SchemaVersao != SchemaVersao {
		return fmt.Errorf("versão não suportada")
	}
	if n := len(p.Token); n < 32 || n > 128 {
		return fmt.Errorf("token de pareamento inválido")
	}
	return primeiroErro(
		uuidCampo("instalacao_uuid", p.InstalacaoUUID),
		textoOpcional("rótulo do aparelho", p.RotuloAparelho),
	)
}
