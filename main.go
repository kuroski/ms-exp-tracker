package main

import (
	"embed"
	"log"

	"github.com/kuroski/ms-exp-tracker/internal/clock"
	"github.com/kuroski/ms-exp-tracker/internal/logger"
	"github.com/kuroski/ms-exp-tracker/internal/services"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "ms-exp-tracker",
		Description: "Experience ExpTracker",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create the main window
	window := app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:         "EXP ExpTracker",
		AlwaysOnTop:   true,
		Frameless:     true,
		DisableResize: true,
		Width:         250,
		Height:        120,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTransparent,
			TitleBar:                application.MacTitleBarDefault,
		},
		BackgroundType: application.BackgroundTypeTransparent,
		// BackgroundColour: application.NewRGB(27, 38, 54),
		URL: "/",
	})

	// Initialize tracker service
	args := services.Args{
		ExpCrawler: services.NewScreenExpCrawler(window),
		Interval:   services.DefaultTickerInterval,
		Clock:      clock.NewRealClock(),
		Logger:     logger.NewLogger(),
	}
	tracker := services.NewExpTracker(args)
	defer tracker.Stop() // ensure clean shutdown

	// Start the tracker in a separate goroutine
	go tracker.Run()

	// Listen for results and forward to frontend
	go func() {
		for result := range tracker.Ch {
			if result.Err != nil {
				log.Printf("Error while tracking: %v", result.Err)
				continue
			}
			log.Printf("Emitting XP update: %+v", result.Stats)
			app.EmitEvent("updateXP", result.Stats)
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
