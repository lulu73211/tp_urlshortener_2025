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

// RunServerCmd représente la commande 'run-server' de Cobra.
// C'est le point d'entrée pour lancer le serveur de l'application.
var RunServerCmd = &cobra.Command{
	Use:   "run-server",
	Short: "Lance le serveur API de raccourcissement d'URLs et les processus de fond.",
	Long: `Cette commande initialise la base de données, configure les APIs,
démarre les workers asynchrones pour les clics et le moniteur d'URLs,
puis lance le serveur HTTP.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO : créer une variable qui stock la configuration chargée globalement via cmd.cfg
		// Ne pas oublier la gestion d'erreur et faire un fatalF
		cfg := cmd2.Cfg
		if cfg == nil {
			log.Fatalf("Configuration not loaded")
		}

		// TODO : Initialiser la connexion à la bBDD
		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Auto-migrate the schema
		err = db.AutoMigrate(&models.Link{}, &models.Click{})
		if err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
		}

		// TODO : Initialiser les repositories.
		// Créez des instances de GormLinkRepository et GormClickRepository.
		linkRepo := repository.NewLinkRepository(db)
		clickRepo := repository.NewClickRepository(db)

		// Laissez le log
		log.Println("Repositories initialisés.")

		// TODO : Initialiser les services métiers.
		// Créez des instances de LinkService et ClickService, en leur passant les repositories nécessaires.
		linkService := services.NewLinkService(linkRepo)
		_ = services.NewClickService(clickRepo) // ClickService pour utilisation future

		// Laissez le log
		log.Println("Services métiers initialisés.")

		// TODO : Configurer le routeur Gin et les handlers API.
		// Passez les services nécessaires aux fonctions de configuration des routes.
		router := gin.Default()
		api.SetupRoutes(router, linkService, cfg.Analytics.BufferSize)

		// TODO : Initialiser le channel ClickEventsChannel (api/handlers) des événements de clic et lancer les workers (StartClickWorkers).
		// Le channel est maintenant initialisé dans handlers.go
		// Start click workers
		workerCount := 2 // Default worker count
		workers.StartClickWorkers(workerCount, api.ClickEventsChannel, clickRepo, linkRepo)

		log.Printf("Channel d'événements de clic initialisé avec un buffer de %d. %d worker(s) de clics démarré(s).",
			cfg.Analytics.BufferSize, workerCount)

		// TODO : Initialiser et lancer le moniteur d'URLs.
		// Utilisez l'intervalle configuré
		monitorInterval := time.Duration(cfg.Monitor.IntervalMinutes) * time.Minute
		urlMonitor := monitor.NewUrlMonitor(linkRepo, monitorInterval) // Le moniteur a besoin du linkRepo et de l'interval

		// TODO Lancez le moniteur dans sa propre goroutine.
		go urlMonitor.Start()

		log.Printf("Moniteur d'URLs démarré avec un intervalle de %v.", monitorInterval)

		log.Println("Routes API configurées.")

		// Créer le serveur HTTP Gin
		serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
		srv := &http.Server{
			Addr:    serverAddr,
			Handler: router,
		}

		// TODO : Démarrer le serveur Gin dans une goroutine anonyme pour ne pas bloquer.
		// Pensez à logger des ptites informations...
		go func() {
			log.Printf("Serveur HTTP démarré sur le port %d", cfg.Server.Port)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Erreur lors du démarrage du serveur: %v", err)
			}
		}()

		// Gére l'arrêt propre du serveur (graceful shutdown).
		// TODO Créez un channel pour les signaux OS (SIGINT, SIGTERM), bufferisé à 1.
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
	// TODO : ajouter la commande
	cmd2.RootCmd.AddCommand(RunServerCmd)
}
