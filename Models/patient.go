package Models

import (
	"gorm.io/gorm"
)

type Patient struct {
	gorm.Model
	Name          string               `json:"name"`
	Phone         string               `json:"phone"`
	Gender        string               `json:"gender"`
	Age           int                  `json:"age"`
	Weight        float64              `json:"weight"`
	Height        float64              `json:"height"`
	Diagnosis     string               `json:"diagnosis"`
	Notes         string               `json:"notes"`
	History       []Appointment        `json:"history"`
	Requests      []AppointmentRequest `json:"requests"`
	OTP           string               `json:"otp"`
	IsVerified    bool                 `json:"is_verified"`
	TreatmentPlan []TreatmentPlan      `json:"treatment_plan"`
}
