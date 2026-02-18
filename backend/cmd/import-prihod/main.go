package main

import (
	"flag"
	"log"

	"github.com/Talonmortem/SHM/db"
)

/*
docker compose up -d postgres
docker compose run --rm backend go run ./cmd/import-prihod -file ./prihod.csv -truncate
*/

func main() {
	filePath := flag.String("file", "./prihod.csv", "Path to prihod.csv file")
	truncate := flag.Bool("truncate", false, "Truncate articles table before import")
	flag.Parse()

	db.ConnectDB()
	defer db.CloseDB()
	db.CreateTables()

	if *truncate {
		if err := db.TruncateArticles(); err != nil {
			log.Fatalf("failed to truncate articles: %v", err)
		}
		log.Println("Articles table truncated")
	}

	result, err := db.ImportArticlesFromCSV(*filePath)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	log.Printf("Import complete: inserted=%d skipped=%d file=%s", result.Inserted, result.Skipped, *filePath)
}
