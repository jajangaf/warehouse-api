package main

import (
	"context"
	"log"
	"time"
	"warehouse-api/pkg/database"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//ctx := context.Background()

	db, err := database.NewConnection(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Succesfully connected to database")
}
