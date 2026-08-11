package main

import (
    "logistic-app-go/routes"
)

func main() {
    // Set up the routes defined in the routes package
    r := routes.SetupRouter()
    
    // Run the server
    r.Run(":8080")
}
