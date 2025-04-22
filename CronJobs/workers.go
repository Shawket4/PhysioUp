package CronJobs

import (
	"PhysioUp/Models"
	"PhysioUp/Whatsapp"
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron"
)

func getNameWithoutDrPrefix(name string) string {
	if len(name) > 4 && name[:4] == "Dr. " {
		return name[4:]
	}
	return name
}

// StartReminderCron starts the cron job to check for appointments and send reminders
func StartReminderCron() *gocron.Scheduler {
	scheduler := gocron.NewScheduler(time.Local)

	scheduler.Every(10).Seconds().Do(func() {
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
		Where("is_completed = ? AND reminder_sent = ? AND date_time::date = ?",
			false,
			false,
			todayDate).
		Find(&appointments)

	if result.Error != nil {
		return fmt.Errorf("failed to query today's appointments: %w", result.Error)
	}

	// Filter appointments that are approximately 3 hours away
	var appointmentsToRemind []Models.Appointment
	fmt.Println(len(appointments))
	for _, appointment := range appointments {
		// Parse the appointment time
		appointmentTime, err := parseDateTime(appointment.DateTime)
		if err != nil {
			log.Printf("Failed to parse appointment time for ID %d: %v", appointment.ID, err)
			continue
		}

		// Create timezone-neutral time objects
		// This effectively strips timezone information for comparison
		appointmentTimeNoTZ := time.Date(
			appointmentTime.Year(),
			appointmentTime.Month(),
			appointmentTime.Day(),
			appointmentTime.Hour(),
			appointmentTime.Minute(),
			appointmentTime.Second(),
			appointmentTime.Nanosecond(),
			time.UTC, // Using UTC as a neutral timezone for both times
		)

		nowNoTZ := time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
			now.Second(),
			now.Nanosecond(),
			time.UTC, // Using UTC as a neutral timezone for both times
		)

		// Calculate time difference with timezone-neutral times
		timeDiff := appointmentTimeNoTZ.Sub(nowNoTZ)

		// Update the time comparison logic to only include upcoming appointments
		if timeDiff > 0 && timeDiff < 3*time.Hour {
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
		// In your SendAppointmentReminders function:
		// Parse the therapist name
		// Create and send reminder message
		message := fmt.Sprintf(
			"🔔 *APPOINTMENT REMINDER* 🔔\\n\\n"+
				"Dear Patient,\\n\\n"+
				"This is a reminder of your upcoming appointment:\\n"+
				"• *Date:* Today\\n"+
				"• *Time:* %s\\n"+
				"• *Therapist:* Dr. %s\\n\\n"+
				"✅ *تذكير بالموعد* ✅\\n\\n"+
				"عزيزي المريض،\\n\\n"+
				"هذا تذكير بموعدك القادم:\\n"+
				"• *التاريخ:* اليوم\\n"+
				"• *الوقت:* %s\\n"+
				"• *دكتور:* %s\\n\\n"+
				"Please arrive on time. If you need to reschedule, please contact us.\\n"+
				"يرجى الحضور في الوقت المحدد. إذا كنت بحاجة إلى إعادة جدولة، يرجى الاتصال بنا.\\n\\n"+
				"Thank you for choosing PhysioUP.\\n"+
				"شكراً لاختيارك PhysioUP.",
			appointmentTime.Format("3:04 PM"),
			getNameWithoutDrPrefix(appointment.TherapistName),
			appointmentTime.Format("3:04 PM"),
			getNameWithoutDrPrefix(appointment.TherapistName),
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
		time.Sleep(time.Second * 5)
	}

	return nil
}

func parseDateTime(dateTimeStr string) (time.Time, error) {
	// Parse with time.UTC to ensure consistent timezone handling
	t, err := time.Parse("2006/01/02 & 3:04 PM", dateTimeStr)
	if err != nil {
		return time.Time{}, err
	}
	// Return the time in UTC to normalize timezone
	return t.UTC(), nil
}
