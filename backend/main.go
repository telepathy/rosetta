package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"rosetta/config"
	"rosetta/database"
	"rosetta/handler"
	"rosetta/middleware"
	"rosetta/service"

	"github.com/gin-gonic/gin"
)

func main() {
	seedDummy := flag.Bool("seed-dummy", false, "Seed 100 dummy tables for testing")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}

	if *seedDummy {
		log.Println("Seeding 100 dummy tables...")
		if err := database.SeedDummyTables(database.DB); err != nil {
			log.Fatalf("Failed to seed dummy data: %v", err)
		}
		log.Println("Dummy tables seeded successfully. Restart without --seed-dummy to run normally.")
		return
	}

	authSvc := service.NewAuthService(database.DB, cfg.JWT)
	authHandler := handler.NewAuthHandler(authSvc)

	userSvc := service.NewUserService(database.DB)
	userHandler := handler.NewUserHandler(userSvc)

	instSvc := service.NewInstanceService(database.DB)
	instHandler := handler.NewInstanceHandler(instSvc)

	dictSvc := service.NewDictService(database.DB)
	dictHandler := handler.NewDictHandler(dictSvc)

	modelSvc := service.NewModelService(database.DB)
	modelHandler := handler.NewModelHandler(modelSvc)

	revEngSvc := service.NewReverseEngService(database.DB)
	revEngHandler := handler.NewReverseEngHandler(revEngSvc)

	vizHandler := handler.NewVizHandler(modelSvc)

	gin.SetMode(cfg.Server.Mode)
	r := gin.Default()

	r.Use(corsMiddleware())
	r.Use(middleware.AuditLog(database.DB))

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.GET("/me", middleware.AuthRequired(cfg.JWT.Secret), authHandler.Me)
			auth.POST("/logout", middleware.AuthRequired(cfg.JWT.Secret), authHandler.Logout)
		}

		handler.RegisterUserRoutes(api, userHandler, cfg.JWT.Secret)
		handler.RegisterInstanceRoutes(api, instHandler, cfg.JWT.Secret)
		handler.RegisterDictRoutes(api, dictHandler, cfg.JWT.Secret)
		handler.RegisterModelRoutes(api, modelHandler, cfg.JWT.Secret)
		handler.RegisterReverseEngRoutes(api, revEngHandler, cfg.JWT.Secret)
		handler.RegisterVizRoutes(api, vizHandler, cfg.JWT.Secret)
	}

	frontendDir := filepath.Join("..", "frontend")
	r.Static("/css", filepath.Join(frontendDir, "css"))
	r.Static("/js", filepath.Join(frontendDir, "js"))
	r.StaticFile("/", filepath.Join(frontendDir, "index.html"))
	r.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join(frontendDir, "index.html"))
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Rosetta server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
