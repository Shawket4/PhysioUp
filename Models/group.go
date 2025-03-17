package Models

import "gorm.io/gorm"

type ClinicGroup struct {
	gorm.Model
	Name string `json:"name" gorm:"unique"`
}
