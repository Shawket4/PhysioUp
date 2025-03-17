package Models

import "gorm.io/gorm"

type TreatmentPlan struct {
	gorm.Model
	Date                 string             `json:"date"`
	SuperTreatmentPlanID uint               `json:"super_treatment_plan_id"`
	SuperTreatmentPlan   SuperTreatmentPlan `json:"super_treatment_plan" gorm:"-"`
	Remaining            uint               `json:"remaining"`
	Discount             float64            `json:"discount"`                         // Discount percentage (e.g., 10 for 10%)
	ReferralID           *uint              `json:"referral_id"  gorm:"default:null"` // Whether this session includes a referral discount
	Referral             Referral           `json:"referral" gorm:"-"`                // Whether this session includes a referral discount
	TotalPrice           float64            `json:"total_price"`
	PatientID            uint               `json:"patient_id"`
	PaymentMethod        string             `json:"payment_method"`
	IsPaid               bool               `json:"is_paid"`
	Appointments         []Appointment
}

type SuperTreatmentPlan struct {
	gorm.Model
	Description    string          `json:"description"`    // Description of the treatment plan
	SessionsCount  uint            `json:"sessions_count"` // List of sessions in the treatment plan
	Price          float64         `json:"price"`          // Price of the session
	TreatmentPlans []TreatmentPlan `json:"treatment_plans"`
	ClinicGroupID  uint            `json:"clinic_group_id"`
}
