// SITPOSSE-7 Central — casca Wails. Todo o núcleo vive em internal/ (ADR-0003).
package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"sitposse7/desktop/app"
	"sitposse7/desktop/internal/config"
	"sitposse7/desktop/internal/syncserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	pasta, err := config.PastaDados()
	if err != nil {
		log.Fatalf("pasta de dados: %v", err)
	}
	central := &app.Central{PastaDados: pasta, Arquivos: arquivosWails{}}
	if err := central.Iniciar(context.Background(), syncserver.PortaPadrao); err != nil {
		log.Fatalf("inicialização: %v", err)
	}
	defer central.Encerrar()

	cartas := &app.CartaService{C: central}
	servicos := []application.Service{application.NewService(&app.SincronizacaoService{C: central})}
	if central.Bloqueio == "" { // banco de geração anterior: só a tela de bloqueio (ADR-0025)
		servicos = append(servicos,
			application.NewService(&app.EstudoService{C: central}),
			application.NewService(&app.ConsultaService{C: central}),
			application.NewService(&app.RelatorioService{C: central}),
			application.NewService(cartas))
	}

	wapp := application.New(application.Options{
		Name:        "SITPOSSE-7 Central",
		Description: "Central de consolidação do SITPOSSE-7 — 7º BPM/PMRO",
		Services:    servicos,
		Assets: application.AssetOptions{
			Handler: rotas(central, cartas, application.AssetFileServerFS(assets)),
		},
	})

	wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "SITPOSSE-7 Central",
		Width:            1280,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(248, 250, 252),
		URL:              "/",
	})

	if err := wapp.Run(); err != nil {
		log.Fatal(err)
	}
}

// rotas serve as fotos (/fotos/), as imagens e cartas guardadas (/cartografia/) e o frontend.
func rotas(central *app.Central, cartas *app.CartaService, frontend http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/fotos/", central.HandlerFotos())
	mux.Handle("/cartografia/", app.HandlerCartografia(cartas))
	mux.Handle("/", frontend)
	return mux
}

// arquivosWails liga as janelas do sistema (Wails) à camada app.
type arquivosWails struct{}

func (arquivosWails) EscolherDestino(nome, descricao, extensao string) (string, error) {
	pasta, _ := os.UserHomeDir()
	if docs := filepath.Join(pasta, "Documents"); dirExiste(docs) {
		pasta = docs
	}
	return application.Get().Dialog.SaveFile().
		SetFilename(nome).
		SetDirectory(pasta).
		AddFilter(descricao, "*"+extensao).
		CanCreateDirectories(true).
		PromptForSingleSelection()
}

func (arquivosWails) Abrir(caminho string) error { return application.Get().Browser.OpenFile(caminho) }

func (arquivosWails) MostrarNaPasta(caminho string) error {
	return application.Get().Env.OpenFileManager(caminho, true)
}

func dirExiste(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
