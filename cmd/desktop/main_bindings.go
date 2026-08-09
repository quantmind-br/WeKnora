//go:build bindings

package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

// Wails' "generate bindings" stage compiles this file separately with -tags bindings, without starting Gin/the database, to avoid depending on a local Postgres.
func main() {
	app := NewApp()
	_ = wails.Run(&options.App{
		Title: "WeKnora Lite",
		Bind:  []interface{}{app},
	})
}
