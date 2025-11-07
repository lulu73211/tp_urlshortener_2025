package cli

import (
	"fmt"
	"log"
	"net/url"
	"os"

	cmd2 "github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// CreateCmd représente la commande 'create'
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Crée une URL courte à partir d'une URL longue.",
	Long: `Cette commande raccourcit une URL longue fournie et affiche le code court généré.

Exemple:
  url-shortener create --url="https://www.google.com/search?q=go+lang"`,
	Run: func(cmd *cobra.Command, args []string) {
		// Récupération du flag --url depuis Cobra
		longURL, _ := cmd.Flags().GetString("url")

		// Valider que le flag --url a été fourni
		if longURL == "" {
			log.Println("Erreur: le flag --url est requis")
			os.Exit(1)
		}

		// Valider que l'URL est bien formée
		_, err := url.Parse(longURL)
		if err != nil {
			log.Printf("Erreur: URL invalide: %v", err)
			os.Exit(1)
		}

		// Charger la configuration chargée globalement via cmd.cfg
		cfg := cmd2.Cfg
		if cfg == nil {
			log.Fatalf("Configuration not loaded")
		}

		// Initialiser la connexion à la BDD
		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Auto-migrate the schema
		err = db.AutoMigrate(&models.Link{}, &models.Click{})
		if err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
		}

		// S'assurer que la connexion est fermée à la fin
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
		}
		defer sqlDB.Close()

		// Initialiser les repositories et services nécessaires
		linkRepo := repository.NewLinkRepository(db)
		linkService := services.NewLinkService(linkRepo)

		// Créer le lien court
		link, err := linkService.CreateLink(longURL)
		if err != nil {
			log.Printf("Erreur lors de la création du lien: %v", err)
			os.Exit(1)
		}

		// Afficher le résultat
		fullShortURL := fmt.Sprintf("%s/%s", cfg.Server.BaseURL, link.Shortcode)
		fmt.Printf("URL courte créée avec succès:\n")
		fmt.Printf("Code court: %s\n", link.Shortcode)
		fmt.Printf("URL longue: %s\n", link.LongURL)
		fmt.Printf("URL complète: %s\n", fullShortURL)
		fmt.Printf("Date de création: %s\n", link.CreatedAt.Format("2006-01-02 15:04:05"))
	},
}

func init() {
	// Définir le flag --url pour la commande create
	CreateCmd.Flags().StringP("url", "u", "", "URL longue à raccourcir")

	// Marquer le flag comme requis
	CreateCmd.MarkFlagRequired("url")

	// Ajouter la commande à RootCmd
	cmd2.RootCmd.AddCommand(CreateCmd)
}
