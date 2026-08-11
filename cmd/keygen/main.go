package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
)

func main() {
	// Generate 32 bytes (256 bits) of random data
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
	}

	// Encode to base64 for easy storage
	encoded := base64.URLEncoding.EncodeToString(secret)
	
	fmt.Println("🔑 Generated JWT Secret Key:")
	fmt.Println(encoded)
	fmt.Printf("\nLength: %d characters\n", len(encoded))
	fmt.Println("\nAdd this to your .env file:")
	fmt.Printf("JWT_SECRET=%s\n", encoded)
}
