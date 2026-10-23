package main

import (
	"log"
	"net/http"
)

func main() {
	router := http.NewServeMux()

	server := http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("Server is running on port 8080")
	log.Fatal(server.ListenAndServe())
}
