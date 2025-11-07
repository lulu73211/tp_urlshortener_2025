package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cmd2 "github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/api"
	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/monitor"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/axellelanca/urlshortener/internal/workers"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite" // Driver SQLite pour GORM
	"gorm.io/gorm"
)

var RunServerCmd = &cobra.Command{
	Use:   "run-server",
	Short: "Lance le serveur API de raccourcissement d'URLs et les processus de fond.",
	Long: `Cette commande initialise la base de données, configure les APIs,
démarre les workers asynchrones pour les clics et le moniteur d'URLs,
puis lance le serveur HTTP.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Gestion d'erreur et faire un fatalF
		cfg := cmd2.Cfg
		if cfg == nil {
			log.Fatalf("Configuration not loaded")
		}

		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Auto-migrate the schema
		err = db.AutoMigrate(&models.Link{}, &models.Click{})
		if err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
		}

		// Instances de GormLinkRepository et GormClickRepository.
		linkRepo := repository.NewLinkRepository(db)
		clickRepo := repository.NewClickRepository(db)

		// Laissez le log
		log.Println("Repositories initialisés.")

		// Créez des instances de LinkService et ClickService, en leur passant les repositories nécessaires.
		linkService := services.NewLinkService(linkRepo)
		_ = services.NewClickService(clickRepo) // ClickService pour utilisation future

		// Laissez le log
		log.Println("Services métiers initialisés.")

		// Passez les services nécessaires aux fonctions de configuration des routes.
		router := gin.Default()
		api.SetupRoutes(router, linkService, cfg.Analytics.BufferSize)

		// Le channel est maintenant initialisé dans handlers.go
		// Start click workers
		workerCount := 2 // Default worker count
		workers.StartClickWorkers(workerCount, api.ClickEventsChannel, clickRepo, linkRepo)

		log.Printf("Channel d'événements de clic initialisé avec un buffer de %d. %d worker(s) de clics démarré(s).",
			cfg.Analytics.BufferSize, workerCount)

		// Utilisez l'intervalle configuré
		monitorInterval := time.Duration(cfg.Monitor.IntervalMinutes) * time.Minute
		urlMonitor := monitor.NewUrlMonitor(linkRepo, monitorInterval) // Le moniteur a besoin du linkRepo et de l'interval

		go urlMonitor.Start()

		log.Printf("Moniteur d'URLs démarré avec un intervalle de %v.", monitorInterval)

		log.Println("Routes API configurées.")

		// Créer le serveur HTTP Gin
		serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
		srv := &http.Server{
			Addr:    serverAddr,
			Handler: router,
		}

		go func() {
			log.Printf("Serveur HTTP démarré sur le port %d", cfg.Server.Port)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Erreur lors du démarrage du serveur: %v", err)
			}
		}()

		// Gére l'arrêt propre du serveur (graceful shutdown).
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // Attendre Ctrl+C ou signal d'arrêt

		// Bloquer jusqu'à ce qu'un signal d'arrêt soit reçu.
		<-quit
		log.Println("Signal d'arrêt reçu. Arrêt du serveur...")

		// Arrêt propre du serveur HTTP avec un timeout.
		log.Println("Arrêt en cours... Donnez un peu de temps aux workers pour finir.")
		time.Sleep(5 * time.Second)

		log.Println("Serveur arrêté proprement.")
	},
}

func init() {
	cmd2.RootCmd.AddCommand(RunServerCmd)
}
