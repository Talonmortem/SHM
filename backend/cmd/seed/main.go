package main

import "github.com/Talonmortem/SHM/db"

/*
docker compose up -d postgres
docker compose run --rm backend go run ./cmd/seed
*/

func main() {
	db.CreateDb()
	db.ConnectDB()
	defer db.CloseDB()
	db.CreateTables()
	db.SeedTestData()
}
