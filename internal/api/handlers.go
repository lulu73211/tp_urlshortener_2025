package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Événement de clic envoyé au worker asynchrone.
type ClickEvent struct {
	ShortCode string
	LongURL   string
	Timestamp time.Time
	IP        string
	UserAgent string
	Referrer  string
}

// Channel global bufferisé utilisé par les workers pour consommer les clics.
var ClickEventsChannel chan ClickEvent

// SetupRoutes configure toutes les routes de l'API Gin et injecte les dépendances nécessaires.
func SetupRoutes(router *gin.Engine, linkService *services.LinkService) {
	// Initialisation du channel des événements de clics.
	if ClickEventsChannel == nil {
		bufferSize := viper.GetInt("analytics.buffer_size")
		if bufferSize <= 0 {
			bufferSize = 100
		}
		ClickEventsChannel = make(chan ClickEvent, bufferSize)
	}

	// Route de Health Check.
	router.GET("/health", HealthCheckHandler)

	// Routes API versionnées.
	api := router.Group("/api/v1")
	{
		api.POST("/links", CreateShortLinkHandler(linkService))
		api.GET("/links/:shortCode/stats", GetLinkStatsHandler(linkService))
	}

	// Route de redirection pour les short codes.
	router.GET("/:shortCode", RedirectHandler(linkService))
}

// HealthCheckHandler gère la route /health pour vérifier l'état du service.
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateLinkRequest représente le corps de la requête JSON pour la création d'un lien.
type CreateLinkRequest struct {
	LongURL string `json:"long_url" binding:"required,url"`
}

// CreateShortLinkHandler gère la création d'une URL courte.
func CreateShortLinkHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateLinkRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		link, err := linkService.CreateLink(req.LongURL)
		if err != nil {
			// Si ton service a des erreurs métiers (URL invalide, etc.), tu peux les tester ici
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create link"})
			return
		}

		baseURL := viper.GetString("server.base_url")
		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}

		c.JSON(http.StatusCreated, gin.H{
			"short_code":     link.Shortcode,
			"long_url":       link.LongURL,
			"full_short_url": baseURL + "/" + link.Shortcode,
		})
	}
}

// RedirectHandler gère la redirection d'une URL courte vers l'URL longue
// et l'enregistrement asynchrone des clics.
func RedirectHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		shortCode := c.Param("shortCode")

		link, err := linkService.GetLinkByShortCode(shortCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
				return
			}
			log.Printf("Error retrieving link for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		clickEvent := ClickEvent{
			ShortCode: shortCode,
			LongURL:   link.LongURL,
			Timestamp: time.Now(),
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Referrer:  c.Request.Referer(),
		}

		// Envoi non bloquant dans le channel bufferisé.
		select {
		case ClickEventsChannel <- clickEvent:
		default:
			log.Printf("Warning: ClickEventsChannel is full, dropping click event for %s.", shortCode)
		}

		c.Redirect(http.StatusFound, link.LongURL)
	}
}

// GetLinkStatsHandler gère la récupération des statistiques pour un lien spécifique.
func GetLinkStatsHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		shortCode := c.Param("shortCode")

		link, totalClicks, err := linkService.GetLinkStats(shortCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
				return
			}
			log.Printf("Error retrieving stats for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"short_code":   link.Shortcode,
			"long_url":     link.LongURL,
			"total_clicks": totalClicks,
		})
	}
}
