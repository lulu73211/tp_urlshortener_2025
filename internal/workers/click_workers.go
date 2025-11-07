package workers

import (
	"log"

	"github.com/axellelanca/urlshortener/internal/api"
	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/repository"
)

// StartClickWorkers lance un pool de goroutines "workers" pour traiter les événements de clic.
func StartClickWorkers(workerCount int, clickEventsChan <-chan api.ClickEvent, clickRepo repository.ClickRepository, linkRepo repository.LinkRepository) {
	log.Printf("Starting %d click worker(s)...", workerCount)
	for i := 0; i < workerCount; i++ {
		go clickWorker(clickEventsChan, clickRepo, linkRepo)
	}
}

// clickWorker traite les événements du channel.
func clickWorker(clickEventsChan <-chan api.ClickEvent, clickRepo repository.ClickRepository, linkRepo repository.LinkRepository) {
	for event := range clickEventsChan {
		// Récupérer le LinkID à partir du ShortCode
		link, err := linkRepo.GetLinkByShortCode(event.ShortCode)
		if err != nil {
			log.Printf("ERROR: Failed to get link for short code %s: %v", event.ShortCode, err)
			continue
		}

		// Conversion api.ClickEvent -> models.Click
		click := models.Click{
			LinkID:    link.ID,
			Timestamp: event.Timestamp,
			UserAgent: event.UserAgent,
			IPAddress: event.IP,
		}

		// Appel à la persistance (il faut que clickRepo ait la méthode CreateClick)
		err = clickRepo.CreateClick(&click)
		if err != nil {
			log.Printf("ERROR: Failed to save click for LinkID %d (UserAgent: %s, IP: %s): %v",
				link.ID, event.UserAgent, event.IP, err)
		} else {
			log.Printf("Click recorded successfully for LinkID %d", link.ID)
		}
	}
}
