package main

import (
	"context"
	"financeirogo/internal/config"
	"financeirogo/internal/database"
	"financeirogo/migrations"
	"flag"
	"log"
	"os"
	"time"
)

func main() {
	apply := flag.Bool("apply", false, "aplica a migration 0001; exige autorização prévia do responsável pelo banco")
	flag.Parse()
	if !*apply || flag.NArg() != 0 {
		log.Print("nenhuma alteração executada; revise migrations/0001_accounts_entries.sql e use -apply somente após autorização")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	raw, err := config.LoadDatabaseURL()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	pool, err := database.Open(ctx, raw)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	err = migrations.Apply(ctx, pool)
	pool.Close()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	log.Print("migration 0001 aplicada ou já registrada com o mesmo conteúdo")
}
