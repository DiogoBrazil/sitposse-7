// SITPOSSE-7 Central — casca Wails. Todo o núcleo vive em internal/ (ADR-0003).
package main

import (
	"context"
	"embed"
	"log"

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
	central := &app.Central{PastaDados: pasta}
	if err := central.Iniciar(context.Background(), syncserver.PortaPadrao); err != nil {
		log.Fatalf("inicialização: %v", err)
	}
	defer central.Encerrar()

	wapp := application.New(application.Options{
		Name:        "SITPOSSE-7 Central",
		Description: "Central de consolidação do SITPOSSE-7 — 7º BPM/PMRO",
		Services: []application.Service{
			application.NewService(&app.SincronizacaoService{C: central}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "SITPOSSE-7 Central",
		Width:            1280,
		Height:           800,
		MinWidth:         1024,
		MinHeight:        700,
		BackgroundColour: application.NewRGB(248, 250, 252),
		URL:              "/",
	})

	if err := wapp.Run(); err != nil {
		log.Fatal(err)
	}
}
