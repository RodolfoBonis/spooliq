package main

import (
	"fmt"
	"os"

	"github.com/RodolfoBonis/spooliq/app"
	"github.com/joho/godotenv"
)

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
