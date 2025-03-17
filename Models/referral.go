package Models

import (
	"gorm.io/gorm"
)

type Referral struct {
	gorm.Model
	Name               string  `json:"name"`
	CashbackPercentage float64 `json:"cashback_percentage"`
	TreatmentPlans     []TreatmentPlan
	ClinicGroupID      uint `json:"clinic_group_id"`
}
