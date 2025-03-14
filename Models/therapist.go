package Models

import (
	"gorm.io/gorm"
)

type Therapist struct {
	gorm.Model
	Name                string               `json:"name"`
	UserID              uint                 `json:"user_id"`
	Phone               string               `json:"phone"`
	Schedule            Schedule             `json:"schedule"`
	AppointmentRequests []AppointmentRequest `json:"requests"`
	// DoctorWorkingHours []DoctorWorkingHour `json:"doctor_working_hours"`
	PhotoUrl string `json:"photo_url"`
	IsDemo   bool   `json:"is_demo"`
	IsFrozen bool   `json:"is_frozen" gorm:"-"`
}

type Schedule struct {
	gorm.Model
	TherapistID uint
	TimeBlocks  []TimeBlock `json:"time_blocks"`
}

type TimeBlock struct {
	gorm.Model
	ScheduleID  uint
	DateTime    string      `json:"date"`
	IsAvailable bool        `json:"is_available"`
	Appointment Appointment `gorm:"constraint:OnDelete:CASCADE;" json:"appointment"`
}

// func CreateDoctorWorkingHours(doctor *Doctor) {
// 	var workingHours []DoctorWorkingHour = []DoctorWorkingHour{{DoctorID: doctor.ID, Time: "07:00 AM"}, {DoctorID: doctor.ID, Time: "07:30 AM"}, {DoctorID: doctor.ID, Time: "08:00 AM"}, {DoctorID: doctor.ID, Time: "08:30 AM"}, {DoctorID: doctor.ID, Time: "09:00 AM"}, {DoctorID: doctor.ID, Time: "09:30 AM"}}
// 	doctor.DoctorWorkingHours = workingHours
// }

func CreateTimeBlock(schedule Schedule, appointment Appointment) TimeBlock {
	return TimeBlock{ScheduleID: schedule.ID, IsAvailable: false, DateTime: appointment.DateTime, Appointment: appointment}
}

func CreateEmptyTimeBlock(schedule Schedule, dateTime string) TimeBlock {
	return TimeBlock{ScheduleID: schedule.ID, IsAvailable: false, DateTime: dateTime}
}
