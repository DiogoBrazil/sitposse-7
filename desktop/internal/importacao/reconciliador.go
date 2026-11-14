package importacao

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RelatorioReconciliacao resume o que o reconciliador fez (integridade.md §Fotos na Central).
type RelatorioReconciliacao struct {
	PartesRemovidas    int      `json:"partes_removidas"`
	OrfasRemovidas     int      `json:"orfas_removidas"`
	RegistrosSemArquivo []string `json:"registros_sem_arquivo"` // ALARME: nunca apagados
}

// IdadeMaximaParte: .part sem atividade há mais tempo que isso é descartado.
const IdadeMaximaParte = 7 * 24 * time.Hour

// Reconciliar limpa restos de recepções interrompidas e confere a invariante 1
// (foto vigente sempre com arquivo). Rodar na inicialização da Central.
func (im *Importador) Reconciliar(ctx context.Context) (RelatorioReconciliacao, error) {
	im.mu.Lock()
	defer im.mu.Unlock()
	var rel RelatorioReconciliacao
	agora := im.agora()

	esperadaEmRecepcao := func(foto string) (bool, error) {
		var n int
		err := im.B.Escrita.QueryRowContext(ctx, `SELECT count(*) FROM fotos_esperadas fe JOIN lotes_recebidos l ON l.uuid = fe.lote_uuid
			WHERE fe.foto_uuid = ? AND l.status = 'recebendo'`, foto).Scan(&n)
		return n > 0, err
	}

	// 1. Staging: .part de foto que nenhum lote em recepção espera, ou parado há 7 dias.
	entradas, err := os.ReadDir(im.dirStaging())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return rel, err
	}
	for _, e := range entradas {
		nome := e.Name()
		if e.IsDir() || !strings.HasSuffix(nome, ".part") {
			continue
		}
		foto := strings.TrimSuffix(nome, ".part")
		info, err := e.Info()
		if err != nil {
			continue
		}
		emRecepcao, err := esperadaEmRecepcao(foto)
		if err != nil {
			return rel, err
		}
		if !emRecepcao || agora.Sub(info.ModTime()) > IdadeMaximaParte {
			if os.Remove(filepath.Join(im.dirStaging(), nome)) == nil {
				rel.PartesRemovidas++
			}
		}
	}

	// 2. Arquivos publicados sem registro (queda entre publicar e gravar a transação):
	//    só são removidos se nenhum lote em recepção ainda os referencia.
	raizFotos := filepath.Join(im.PastaDados, "fotos")
	err = filepath.WalkDir(raizFotos, func(caminho string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".jpg") {
			return nil
		}
		foto := strings.TrimSuffix(d.Name(), ".jpg")
		var n int
		if err := im.B.Escrita.QueryRowContext(ctx, `SELECT count(*) FROM fotografias WHERE uuid = ? AND caminho_relativo IS NOT NULL`, foto).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		emRecepcao, err := esperadaEmRecepcao(foto)
		if err != nil {
			return err
		}
		if !emRecepcao && os.Remove(caminho) == nil {
			rel.OrfasRemovidas++
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return rel, err
	}

	// 3. Registros vigentes sem arquivo: alarme (nunca apagar o registro).
	rows, err := im.B.Escrita.QueryContext(ctx, `SELECT uuid, caminho_relativo FROM fotografias WHERE excluido_em IS NULL`)
	if err != nil {
		return rel, err
	}
	defer rows.Close()
	for rows.Next() {
		var uuid, rel2 string
		if err := rows.Scan(&uuid, &rel2); err != nil {
			return rel, err
		}
		if _, err := os.Stat(filepath.Join(im.PastaDados, filepath.FromSlash(rel2))); err != nil {
			rel.RegistrosSemArquivo = append(rel.RegistrosSemArquivo, uuid)
		}
	}
	return rel, rows.Err()
}
