package Models

import (
	"math/rand"

	"gorm.io/gorm"
)

type Appointment struct {
	gorm.Model
	DateTime        string `json:"date_time"`
	TimeBlockID     uint
	TherapistID     uint    `json:"therapist_id"`
	TherapistName   string  `json:"therapist_name"`
	PatientName     string  `json:"patient_name"`
	PatientID       uint    `json:"patient_id"`
	Price           float64 `json:"price"`
	IsCompleted     bool    `json:"is_completed"`
	IsPaid          bool    `json:"is_paid"`
	PaymentMethod   string  `json:"payment_method"`
	Notes           string  `json:"notes"`
	TreatmentPlanID *uint   `json:"treatment_plan_id" gorm:"default:null"`
	ReminderSent    bool    `json:"reminder_sent"`
}

type AppointmentRequest struct {
	gorm.Model
	DateTime                      string `json:"date_time"`
	TherapistID                   uint   `json:"therapist_id"`
	TherapistName                 string `json:"therapist_name"`
	PatientName                   string `json:"patient_name"`
	PatientID                     uint   `json:"patient_id"`
	PhoneNumber                   string `json:"phone_number"`
	SuperTreatmentPlanDescription string `json:"super_treatment_plan_description"`
	IsExisting                    bool   `json:"is_existing" gorm:"-"`
}

func (patient *Patient) GenerateOTPToken(count int) {
	var possibleCharacters = []rune("1234567890")

	token := make([]rune, count)
	for index := range token {
		token[index] = possibleCharacters[rand.Intn(len(possibleCharacters))]
	}
	patient.OTP = string(token)
}
