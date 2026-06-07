package main

import (
	"context"
	"fmt"
	"log"
	"os"

	delivery "github.com/flow-forger/flow-forger/backend/delivery/http"
	"github.com/flow-forger/flow-forger/backend/delivery/http/controllers"
	"github.com/flow-forger/flow-forger/backend/domain"
	"github.com/flow-forger/flow-forger/backend/repository"
	"github.com/flow-forger/flow-forger/backend/usecase"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load .env variables if present
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	// Initialize PostgreSQL Connection
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "flowforge"
	}
	dbSSLMode := os.Getenv("DB_SSLMODE")
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	db, err := repository.InitDB(postgres.Open(dsn))
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 2. Seed Initial Test Data (if not already seeded)
	seedData(db)

	// 3. Initialize Repositories
	userRepo := repository.NewUserRepository(db)
	wfRepo := repository.NewWorkflowRepository(db)
	runRepo := repository.NewRunRepository(db)

	// 4. Initialize Controllers
	hub := controllers.NewEventHub()
	analyzer := usecase.NewOpenRouterAnalyzer()
	authCtrl := controllers.NewAuthController(db, userRepo)
	wfCtrl := controllers.NewWorkflowController(wfRepo, runRepo, hub, analyzer)
	runCtrl := controllers.NewRunController(runRepo)
	healthCtrl := controllers.NewHealthController(runRepo)
	sseCtrl := controllers.NewSSEController(hub)

	// 5. Setup Fiber Application
	app := fiber.New()

	// 6. Setup Router Routing
	delivery.SetupRoutes(app, authCtrl, wfCtrl, runCtrl, healthCtrl, sseCtrl)

	// 7. Start Listening
	log.Println("FlowForge backend starting on :8080...")
	if err := app.Listen(":8080"); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func seedData(db *gorm.DB) {
	// Create Tenant A
	var count int64
	db.Model(&domain.Tenant{}).Where("id = ?", "tenant-a").Count(&count)
	if count == 0 {
		tenant := &domain.Tenant{
			ID:       "tenant-a",
			TenantID: "tenant-a",
			Name:     "Tenant A Corp",
		}
		if err := db.Create(tenant).Error; err != nil {
			log.Printf("Failed to seed tenant-a: %v", err)
		} else {
			log.Println("Seeded default tenant: tenant-a")
		}
	}

	// Create User editor@tenant-a.com
	db.Model(&domain.User{}).Where("email = ?", "editor@tenant-a.com").Count(&count)
	if count == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		user := &domain.User{
			ID:           "user-editor",
			TenantID:     "tenant-a",
			Email:        "editor@tenant-a.com",
			PasswordHash: string(hashedPassword),
			Role:         "Editor",
		}
		if err := db.Create(user).Error; err != nil {
			log.Printf("Failed to seed user: %v", err)
		} else {
			log.Println("Seeded default user: editor@tenant-a.com / password123")
		}
	}

	// Create default Workflow wf-default
	db.Model(&domain.Workflow{}).Where("id = ?", "wf-default").Count(&count)
	if count == 0 {
		definition := `{
			"steps": [
				{
					"id": "fetch-data",
					"type": "delay",
					"config": 1000000000
				},
				{
					"id": "transform-data",
					"type": "default",
					"depends_on": ["fetch-data"]
				},
				{
					"id": "load-data",
					"type": "default",
					"depends_on": ["transform-data"]
				}
			]
		}`
		
		ctx := context.WithValue(context.Background(), "tenant_id", "tenant-a")
		wf := &domain.Workflow{
			ID:         "wf-default",
			TenantID:   "tenant-a",
			Version:    1,
			Name:       "Default ETL Pipeline",
			Definition: definition,
		}
		if err := db.WithContext(ctx).Create(wf).Error; err != nil {
			log.Printf("Failed to seed workflow: %v", err)
		} else {
			log.Println("Seeded default workflow: wf-default")
		}
	}
}
