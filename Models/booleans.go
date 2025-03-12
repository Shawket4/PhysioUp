package Models

import "gorm.io/gorm"

type Boolean struct {
	gorm.Model
	Key   string `json:"key"`
	Value bool   `json:"value"`
}
