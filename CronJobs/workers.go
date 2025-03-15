package CronJobs

import (
	"PhysioUp/Models"
	"PhysioUp/Whatsapp"
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron"
	"gorm.io/gorm"
)

// AppointmentReminder handles sending reminder messages for upcoming appointments
type AppointmentReminder struct {
	DB *gorm.DB
}

// NewAppointmentReminder creates a new appointment reminder service
func NewAppointmentReminder(db *gorm.DB) *AppointmentReminder {
	return &AppointmentReminder{
		DB: db,
	}
}

// StartReminderCron starts the cron job to check for appointments and send reminders
func (ar *AppointmentReminder) StartReminderCron() *gocron.Scheduler {
	scheduler := gocron.NewScheduler(time.Local)

	// Run every 15 minutes to check for appointments that need reminders
	scheduler.Every(1).Minutes().Do(func() {
		log.Println("Running appointment reminder check...")
		if err := ar.SendAppointmentReminders(); err != nil {
			log.Printf("Error sending appointment reminders: %v", err)
		}
	})

	scheduler.StartAsync()
	log.Println("Appointment reminder cron job started")

	return scheduler
}

func (ar *AppointmentReminder) SendAppointmentReminders() error {
	// Current time
	now := time.Now()

	startWindow := now.Add(2*time.Hour + 53*time.Minute)
	endWindow := now.Add(3*time.Hour + 7*time.Minute)

	var appointments []Models.Appointment

	result := ar.DB.Joins("JOIN patients ON appointments.patient_id = patients.id").
		Where("appointments.is_completed = ? AND appointments.date_time BETWEEN ? AND ?",
			false,
			formatDateTime(startWindow),
			formatDateTime(endWindow)).
		Find(&appointments)

	if result.Error != nil {
		return fmt.Errorf("failed to query upcoming appointments: %w", result.Error)
	}

	for _, appointment := range appointments {
		var patient Models.Patient
		if err := ar.DB.First(&patient, appointment.PatientID).Error; err != nil {
			log.Printf("Failed to find patient for appointment ID %d: %v", appointment.ID, err)
			continue
		}

		if !patient.IsVerified || patient.Phone == "" {
			continue
		}

		appointmentTime, err := parseDateTime(appointment.DateTime)
		if err != nil {
			log.Printf("Failed to parse appointment time for ID %d: %v", appointment.ID, err)
			continue
		}

		message := fmt.Sprintf(
			"Reminder: You have an appointment with %s today at %s (in 3 hours). "+
				"Please arrive 10 minutes early. If you need to reschedule, please contact us.",
			appointment.TherapistName,
			appointmentTime.Format("3:04 PM"),
		)

		if err := Whatsapp.SendMessage(patient.Phone, message); err != nil {
			log.Printf("Failed to send reminder to patient %s: %v", patient.Name, err)
			continue
		}

		log.Printf("Reminder sent to %s for appointment at %s", patient.Name, appointment.DateTime)
	}

	return nil
}

func parseDateTime(dateTimeStr string) (time.Time, error) {
	return time.Parse("2006/01/02 & 3:04 PM", dateTimeStr)
}

func formatDateTime(t time.Time) string {
	return t.Format("2006/01/02 & 3:04 PM")
}
