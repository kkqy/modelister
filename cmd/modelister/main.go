package main

import (
	"log"
	"net/http"

	"modelister/internal/auth"
	"modelister/internal/config"
	"modelister/internal/models"
	"modelister/internal/providers"
	"modelister/internal/server"
	"modelister/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置错误 / config error: %v", err)
	}

	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("数据库错误 / database error: %v", err)
	}
	defer db.Close()

	providerRepo := providers.NewRepository(db)
	modelRepo := models.NewRepository(db)
	syncService := models.NewSyncService(providerRepo, modelRepo, http.DefaultClient)

	router := server.New(
		auth.NewManager(cfg.AdminUsername, cfg.AdminPassword, cfg.SessionSecret),
		providers.NewHandler(providerRepo),
		models.NewHandler(modelRepo, syncService),
	)

	log.Printf("Modelister listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, router); err != nil {
		log.Fatalf("HTTP 服务错误 / http server error: %v", err)
	}
}
