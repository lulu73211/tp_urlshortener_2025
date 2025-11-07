package workers

import (
	"log"

	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/repository"
)

// StartClickWorkers lance un pool de goroutines "workers" pour traiter les événements de clic.
func StartClickWorkers(workerCount int, clickEventsChan <-chan models.ClickEvent, clickRepo repository.ClickRepository) {
	log.Printf("Starting %d click worker(s)...", workerCount)
	for i := 0; i < workerCount; i++ {
		go clickWorker(clickEventsChan, clickRepo)
	}
}

// clickWorker traite les événements du channel.
func clickWorker(clickEventsChan <-chan models.ClickEvent, clickRepo repository.ClickRepository) {
	for event := range clickEventsChan {
		// Conversion ClickEvent -> models.Click
		click := models.Click{
			LinkID:    event.LinkID,
			Timestamp: event.Timestamp,
			UserAgent: event.UserAgent,
			IPAddress: event.IPAddress,
		}

		// Appel à la persistance (il faut que clickRepo ait la méthode CreateClick)
		err := clickRepo.CreateClick(&click)
		if err != nil {
			log.Printf("ERROR: Failed to save click for LinkID %d (UserAgent: %s, IP: %s): %v",
				event.LinkID, event.UserAgent, event.IPAddress, err)
		} else {
			log.Printf("Click recorded successfully for LinkID %d", event.LinkID)
		}
	}
}
