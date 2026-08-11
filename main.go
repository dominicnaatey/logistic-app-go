package main

import (
	"logistic-app-go/config"
	"logistic-app-go/routes"
)

func main() {
	// Connect to MongoDB
	config.ConnectDB("mongodb://localhost:27017", "logistic-app")

	// Set up routes
	r := routes.SetupRouter()

	// Run the server
	r.Run(":8080")
}
