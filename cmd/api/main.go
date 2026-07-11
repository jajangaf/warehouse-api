package main

import (
	"context"
	"log"
	"warehouse-api/pkg/database"
)

func main() {
	ctx := context.Background()

	db, err := database.NewConnection(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Succesfully connected to database")
}
