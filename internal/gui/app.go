// Package gui provides the system tray icon and configuration window
// using the Fyne UI toolkit.
package gui

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// App holds the Fyne application state.
type App struct {
	fyneApp fyne.App
	window  fyne.Window
}

// New creates a new GUI App.
func New() *App {
	return &App{}
}

// Run starts the Fyne application and shows the main window.
// Must be called from the main goroutine.
func (a *App) Run() {
	a.fyneApp = app.New()
	a.window = a.fyneApp.NewWindow("NetInput Share")
	a.window.SetContent(a.buildUI())
	a.window.Resize(fyne.NewSize(400, 300))
	slog.Info("gui: starting")
	a.window.ShowAndRun()
}

func (a *App) buildUI() fyne.CanvasObject {
	title := widget.NewLabel("NetInput Share")
	status := widget.NewLabel("Status: Running")
	screens := widget.NewLabel("Screens: 4 configured")

	switchBtn := widget.NewButton("Switch to Next Screen", func() {
		slog.Info("gui: manual switch triggered")
	})

	return container.NewVBox(title, status, screens, switchBtn)
}
