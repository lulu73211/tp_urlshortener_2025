package models

import "time"

type Link struct {
	ID        uint      `gorm:"primaryKey"`           // Clé primaire
	Shortcode string    `gorm:"unique;index;size:10"` // Code court unique, indexé pour des recherches rapides
	LongURL   string    `gorm:"not null"`             // URL complète du lien
	CreatedAt time.Time `gorm:"autoCreateTime"`       // Horodatage de la création du lien
}
