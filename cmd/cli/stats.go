package cli

import (
	"fmt"
	"log"
	"os"

	cmd2 "github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/spf13/cobra"

	"gorm.io/driver/sqlite" // Driver SQLite pour GORM
	"gorm.io/gorm"
)

// StatsCmd représente la commande 'stats'
var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Affiche les statistiques d'un lien court",
	Long: `Cette commande affiche les statistiques détaillées pour un code court donné,
incluant l'URL longue associée et le nombre total de clics.

Exemple d'utilisation:
  url-shortener stats --code abc123`,
	Run: func(cmd *cobra.Command, args []string) {
		// Récupération du flag --code depuis Cobra.
		shortCodeFlag, _ := cmd.Flags().GetString("code")

		if shortCodeFlag == "" {
			log.Println("Erreur: le flag --code est requis")
			os.Exit(1)
		}

		cfg := cmd2.Cfg
		if cfg == nil {
			log.Fatalf("Configuration not loaded")
		}

		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
		}

		defer sqlDB.Close()

		linkRepo := repository.NewLinkRepository(db)
		linkService := services.NewLinkService(linkRepo)

		link, totalClicks, err := linkService.GetLinkStats(shortCodeFlag)
		if err != nil {
			log.Printf("Erreur lors de la récupération des statistiques: %v", err)
			os.Exit(1)
		}

		fmt.Printf("Statistiques pour le code court: %s\n", link.Shortcode)
		fmt.Printf("URL longue: %s\n", link.LongURL)
		fmt.Printf("Date de création: %s\n", link.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Nombre total de clics: %d\n", totalClicks)
	},
}

func init() {
	StatsCmd.Flags().StringP("code", "c", "", "Code court pour lequel afficher les statistiques")

	StatsCmd.MarkFlagRequired("code")

	cmd2.RootCmd.AddCommand(StatsCmd)
}
