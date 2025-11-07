package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	//"net/url" // TODO: À décommenter quand tu valideras l'URL
	//"os"      // TODO: À décommenter quand tu feras des os.Exit()
	//"github.com/axellelanca/urlshortener/cmd" // TODO: À décommenter si utilisé
	//"github.com/axellelanca/urlshortener/internal/repository" // TODO: idem
	//"github.com/axellelanca/urlshortener/internal/services"   // TODO: idem
	//"gorm.io/driver/sqlite" // TODO: À décommenter si utilisé
	//"gorm.io/gorm"          // TODO: À décommenter si utilisé
)

// CreateCmd représente la commande 'create'
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Crée une URL courte à partir d'une URL longue.",
	Long: `Cette commande raccourcit une URL longue fournie et affiche le code court généré.

Exemple:
  url-shortener create --url="https://www.google.com/search?q=go+lang"`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implémenter la logique ici

		// Exemple pour compiler sans erreur :
		var cfg struct {
			Server struct {
				BaseURL string
			}
		}
		cfg.Server.BaseURL = "http://localhost:8080"

		type Link struct {
			ShortCode string
		}
		link := Link{ShortCode: "exemple"}

		fullShortURL := fmt.Sprintf("%s/%s", cfg.Server.BaseURL, link.ShortCode)
		fmt.Printf("URL courte créée avec succès:\n")
		fmt.Printf("Code: %s\n", link.ShortCode)
		fmt.Printf("URL complète: %s\n", fullShortURL)
	},
}

func init() {
	// À compléter plus tard
}
