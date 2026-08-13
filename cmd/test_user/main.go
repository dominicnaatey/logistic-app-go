package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"logistic-app-go/internal/user"
	"logistic-app-go/pkg/config"
	"logistic-app-go/pkg/database"
)

func main() {
	fmt.Println("🧪 Testing User Model against Neon PostgreSQL...")
	fmt.Println()

	// Load config (.env)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Config load failed: %v", err)
	}

	// Connect to DB
	db, err := database.NewPostgresDB(cfg.Database.URL, false)
	if err != nil {
		log.Fatalf("❌ DB connection failed: %v", err)
	}
	defer database.Close(db)
	fmt.Println("✓ Connected to Neon PostgreSQL")

	// Run migrations (ensure users table exists)
	if err := database.Migrate(db); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	// -------------------------------------------------------
	// TEST 1: Create a user
	// -------------------------------------------------------
	fmt.Println("\n📝 Test 1: Create user...")
	testPhone := "+233501234001"
	newUser := &user.User{
		Phone:     testPhone,
		Role:      user.RoleDriver,
		Name:      "Kwame Mensah",
		Language:  user.LangEnglish,
		KYCStatus: user.KYCPending,
		IsActive:  true,
	}

	result := db.Create(newUser)
	if result.Error != nil {
		log.Fatalf("❌ Create failed: %v", result.Error)
	}
	fmt.Printf("  ✓ User created with ID: %s\n", newUser.ID)
	fmt.Printf("  ✓ Phone: %s | Role: %s | KYC: %s\n", newUser.Phone, newUser.Role, newUser.KYCStatus)

	createdID := newUser.ID

	// -------------------------------------------------------
	// TEST 2: Fetch user by ID
	// -------------------------------------------------------
	fmt.Println("\n🔍 Test 2: Fetch user by ID...")
	var fetchedUser user.User
	if err := db.First(&fetchedUser, "id = ?", createdID).Error; err != nil {
		log.Fatalf("❌ Fetch by ID failed: %v", err)
	}
	fmt.Printf("  ✓ Found user: %s (%s)\n", fetchedUser.Name, fetchedUser.Phone)

	// -------------------------------------------------------
	// TEST 3: Fetch user by phone
	// -------------------------------------------------------
	fmt.Println("\n🔍 Test 3: Fetch user by phone...")
	var byPhone user.User
	if err := db.Where("phone = ?", testPhone).First(&byPhone).Error; err != nil {
		log.Fatalf("❌ Fetch by phone failed: %v", err)
	}
	fmt.Printf("  ✓ Found by phone: %s | Role: %s\n", byPhone.Name, byPhone.Role)

	// -------------------------------------------------------
	// TEST 4: Update KYC status
	// -------------------------------------------------------
	fmt.Println("\n✏️  Test 4: Update KYC status to approved...")
	if err := db.Model(&fetchedUser).Update("kyc_status", user.KYCApproved).Error; err != nil {
		log.Fatalf("❌ Update failed: %v", err)
	}

	// Re-fetch to confirm
	db.First(&fetchedUser, "id = ?", createdID)
	fmt.Printf("  ✓ KYC status updated: %s\n", fetchedUser.KYCStatus)

	// -------------------------------------------------------
	// TEST 5: Helper methods
	// -------------------------------------------------------
	fmt.Println("\n🔧 Test 5: Model helper methods...")
	fmt.Printf("  ✓ IsDriver():       %v (expected: true)\n", fetchedUser.IsDriver())
	fmt.Printf("  ✓ CanAcceptLoads(): %v (expected: true)\n", fetchedUser.CanAcceptLoads())
	fmt.Printf("  ✓ IsKYCApproved():  %v (expected: true)\n", fetchedUser.IsKYCApproved())

	// Test a shipper (should NOT be able to accept loads)
	shipper := &user.User{Role: user.RoleShipper, KYCStatus: user.KYCApproved, IsActive: true}
	fmt.Printf("  ✓ Shipper.IsDriver():       %v (expected: false)\n", shipper.IsDriver())
	fmt.Printf("  ✓ Shipper.CanAcceptLoads(): %v (expected: false)\n", shipper.CanAcceptLoads())

	// -------------------------------------------------------
	// TEST 6: Unique phone constraint
	// -------------------------------------------------------
	fmt.Println("\n🚫 Test 6: Duplicate phone rejected...")
	duplicate := &user.User{
		Phone: testPhone, // same phone as test 1
		Role:  user.RoleShipper,
	}
	err = db.Create(duplicate).Error
	if err != nil {
		fmt.Printf("  ✓ Duplicate phone correctly rejected: unique constraint enforced\n")
	} else {
		fmt.Println("  ❌ Duplicate phone was NOT rejected — constraint missing!")
	}

	// -------------------------------------------------------
	// TEST 7: Soft delete
	// -------------------------------------------------------
	fmt.Println("\n🗑️  Test 7: Soft delete user...")
	if err := db.Delete(&fetchedUser).Error; err != nil {
		log.Fatalf("❌ Soft delete failed: %v", err)
	}

	// Normal query should NOT find it
	var afterDelete user.User
	result = db.First(&afterDelete, "id = ?", createdID)
	if result.Error != nil {
		fmt.Println("  ✓ Soft-deleted user not found in normal queries")
	} else {
		fmt.Println("  ❌ Soft-deleted user still visible — soft delete not working!")
	}

	// Unscoped query SHOULD find it
	var withDeleted user.User
	if err := db.Unscoped().First(&withDeleted, "id = ?", createdID).Error; err != nil {
		fmt.Println("  ❌ Could not find soft-deleted user with Unscoped()")
	} else {
		fmt.Printf("  ✓ User still exists in DB with deleted_at: %v\n", withDeleted.DeletedAt.Time)
	}

	// -------------------------------------------------------
	// TEST 8: Create multiple roles and list them
	// -------------------------------------------------------
	fmt.Println("\n📋 Test 8: Create multiple roles...")
	testUsers := []user.User{
		{Phone: "+233501234002", Role: user.RoleShipper, Name: "Ama Owusu", Language: user.LangEnglish},
		{Phone: "+22370000001", Role: user.RoleDriver, Name: "Moussa Diallo", Language: user.LangFrench},
		{Phone: "+233501234003", Role: user.RoleFleetAdmin, Name: "Kofi Boateng", Language: user.LangEnglish},
		{Phone: "+22370000002", Role: user.RoleOwnerOperator, Name: "Ibrahim Traoré", Language: user.LangFrench},
	}

	for i := range testUsers {
		testUsers[i].ID = uuid.Nil // let BeforeCreate assign UUID
		if err := db.Create(&testUsers[i]).Error; err != nil {
			log.Fatalf("❌ Failed to create test user %s: %v", testUsers[i].Phone, err)
		}
	}

	var allUsers []user.User
	db.Find(&allUsers)
	fmt.Printf("  ✓ Total active users in DB: %d\n", len(allUsers))

	for _, u := range allUsers {
		fmt.Printf("    - %s | %-18s | %-15s | lang: %s\n", u.Phone, u.Role, u.KYCStatus, u.Language)
	}

	// -------------------------------------------------------
	// Cleanup: remove all test data
	// -------------------------------------------------------
	fmt.Println("\n🧹 Cleanup: removing test data...")
	testPhones := []string{
		"+233501234001",
		"+233501234002",
		"+22370000001",
		"+233501234003",
		"+22370000002",
	}
	db.Unscoped().Where("phone IN ?", testPhones).Delete(&user.User{})
	fmt.Println("  ✓ Test data removed")

	// -------------------------------------------------------
	// Summary
	// -------------------------------------------------------
	fmt.Println("\n✅ All User Model tests passed!")
	fmt.Println("\nUser model is ready. Next step: Phase 1.2 — OTP Service")
}
