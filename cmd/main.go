package main

import (
	"store"
)

func main() {
	const Host = "127.0.0.1"
	const Port = "8080"
	const MaxKeyValues = 30

	db := store.NewDatabase()
	service := store.NewService(db, MaxKeyValues)
	service.StartServer(Host, Port)
}
