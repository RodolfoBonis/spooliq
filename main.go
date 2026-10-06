package main

import (
	"fmt"
	"os"

	"github.com/RodolfoBonis/spooliq/app"
	"github.com/joho/godotenv"
)

// @title SpoolIQ API
// @version 2.8.0
// @description SpoolIq calcula o preço real das suas impressões 3D: filamento multi-cor (g/m), energia (kWh + bandeira), desgaste, overhead e mão-de-obra. Gera pacotes (só impressão, ajustes, modelagem), exporta PDF/CSV e guarda materiais.
// @BasePath /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load .env before FX starts so env vars are available to all providers
	// (e.g., go-otel-agent reads OTEL_* vars at Agent creation time)
	env := os.Getenv("ENV")
	if env != "production" && env != "staging" {
		filename := fmt.Sprintf(".env.%s", env)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			filename = ".env"
		}
		_ = godotenv.Load(filename)
	}

	app.NewFxApp().Run()
}
