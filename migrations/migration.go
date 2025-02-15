package main

import (
	"main.go/infra"
	"main.go/models"
)

func main() {
	infra.Initialize()
	db := infra.SetupDB()

	if err := db.AutoMigrate(&models.Score{}, &models.GameState{}); err != nil {
		panic("Failed to migrate database")
	}
}
