package CronJobs

import (
	"PhysioUp/Models"
	"PhysioUp/Whatsapp"
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron"
)

// StartReminderCron starts the cron job to check for appointments and send reminders
func StartReminderCron() *gocron.Scheduler {
	scheduler := gocron.NewScheduler(time.Local)

	// Run every 15 minutes to check for appointments that need reminders
	scheduler.Every(1).Minutes().Do(func() {
		log.Println("Running appointment reminder check...")
		if err := SendAppointmentReminders(); err != nil {
			log.Printf("Error sending appointment reminders: %v", err)
		}
	})

	scheduler.StartAsync()
	log.Println("Appointment reminder cron job started")

	return scheduler
}
func SendAppointmentReminders() error {
	// Current time
	now := time.Now()

	// Format dates for the query
	todayDate := now.Format("2006/01/02")

	// Get all of today's non-completed appointments that haven't had reminders sent yet
	var appointments []Models.Appointment
	result := Models.DB.Model(&Models.Appointment{}).
		Where("is_completed = ? AND reminder_sent = ? AND date_time LIKE ?",
			false,
			false,
			todayDate+"%").
		Find(&appointments)

	if result.Error != nil {
		return fmt.Errorf("failed to query today's appointments: %w", result.Error)
	}
	fmt.Println(appointments)
	// Filter appointments that are approximately 3 hours away
	var appointmentsToRemind []Models.Appointment
	for _, appointment := range appointments {
		// Parse the appointment time
		appointmentTime, err := parseDateTime(appointment.DateTime)
		if err != nil {
			log.Printf("Failed to parse appointment time for ID %d: %v", appointment.ID, err)
			continue
		}

		// Calculate time difference
		timeDiff := appointmentTime.Sub(now)

		// Check if appointment is approximately 3 hours away (within the window)
		if timeDiff >= 2*time.Hour+53*time.Minute && timeDiff <= 3*time.Hour+7*time.Minute {
			appointmentsToRemind = append(appointmentsToRemind, appointment)
		}
	}

	// Process each appointment that needs a reminder
	for _, appointment := range appointmentsToRemind {
		var patient Models.Patient
		if err := Models.DB.First(&patient, appointment.PatientID).Error; err != nil {
			log.Printf("Failed to find patient for appointment ID %d: %v", appointment.ID, err)
			continue
		}

		// Skip if patient not verified or no phone number
		if !patient.IsVerified || patient.Phone == "" {
			continue
		}

		// Parse appointment time for formatting in the message
		appointmentTime, err := parseDateTime(appointment.DateTime)
		if err != nil {
			log.Printf("Failed to parse appointment time for ID %d: %v", appointment.ID, err)
			continue
		}

		// Create and send reminder message
		message := fmt.Sprintf(
			"Reminder: You have an appointment with Dr. %s today at %s (in 3 hours). "+
				"Please arrive on time. If you need to reschedule, please contact us.",
			appointment.TherapistName,
			appointmentTime.Format("3:04 PM"),
		)

		if err := Whatsapp.SendMessage(patient.Phone, message); err != nil {
			log.Printf("Failed to send reminder to patient %s: %v", patient.Name, err)
			continue
		}

		// Update the appointment to mark reminder as sent
		appointment.ReminderSent = true
		if err := Models.DB.Save(&appointment).Error; err != nil {
			log.Printf("Failed to update reminder sent status for appointment ID %d: %v", appointment.ID, err)
			// Continue anyway since the message was already sent
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
