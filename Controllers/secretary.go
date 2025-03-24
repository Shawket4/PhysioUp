package Controllers

import (
	"PhysioUp/FirebaseMessaging"
	"PhysioUp/Models"
	"PhysioUp/SSE"
	"PhysioUp/Utils/Token"
	"PhysioUp/Whatsapp"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func AcceptAppointment(c *gin.Context) {
	var input struct {
		AppointmentRequestID uint               `json:"appointment_request_id"`
		Extra                Models.Appointment `json:"extra"`
		// TreatmentPlan        Models.TreatmentPlan `json:"treatment_plan"`
	}

	if err := c.ShouldBindBodyWith(&input, binding.JSON); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// input.TreatmentPlan.Date = time.Now().Format("2006-01-02")
	// Start a transaction
	tx := Models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback() // Rollback the transaction in case of panic
		}
	}()

	// Fetch the appointment request
	var appointmentRequest Models.AppointmentRequest
	if err := tx.Model(&Models.AppointmentRequest{}).Where("id = ?", input.AppointmentRequestID).First(&appointmentRequest).Error; err != nil {
		log.Println(err.Error())
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Appointment request not found"})
		return
	}

	// Populate appointment details
	var appointment Models.Appointment = input.Extra
	appointment.PatientName = appointmentRequest.PatientName
	appointment.TherapistID = appointmentRequest.TherapistID
	appointment.TherapistName = appointmentRequest.TherapistName
	appointment.PatientID = appointmentRequest.PatientID
	appointment.TreatmentPlanID = nil
	appointment.ClinicGroupID = appointmentRequest.ClinicGroupID
	appointmentTime, err := time.Parse("2006/01/02 & 3:04 PM", appointmentRequest.DateTime)
	if appointmentTime.After(time.Now()) {
		appointment.ReminderSent = true
	}
	// Check therapist's schedule for conflicts
	var therapist Models.Therapist
	if err := tx.Model(&Models.Therapist{}).Where("id = ?", appointment.TherapistID).Preload("Schedule.TimeBlocks").Find(&therapist).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Therapist not found"})
		return
	}

	therapist.Schedule.TherapistID = appointment.TherapistID

	for _, timeblock := range therapist.Schedule.TimeBlocks {
		if timeblock.DateTime == appointment.DateTime {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Time block already booked"})
			return
		}
	}

	// Create a new time block
	timeBlock := Models.CreateTimeBlock(therapist.Schedule, appointment)
	therapist.Schedule.TimeBlocks = append(therapist.Schedule.TimeBlocks, timeBlock)

	if err := tx.Model(&therapist.Schedule).Where("id = ?", therapist.Schedule.ID).Association("TimeBlocks").Replace(&therapist.Schedule.TimeBlocks); err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to update therapist schedule"})
		return
	}

	// Associate the appointment with the time block
	if err := tx.Model(&therapist.Schedule.TimeBlocks[len(therapist.Schedule.TimeBlocks)-1]).Association("Appointment").Replace(&appointment); err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to associate appointment with time block"})
		return
	}

	// Delete the appointment request
	if err := tx.Delete(&appointmentRequest, "id = ?", appointmentRequest.ID).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to delete appointment request"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appointment registered successfully"})
	user_id, err := Token.ExtractTokenID(c)
	if err != nil {
		log.Println(err)
	}
	fcms, _ := Models.GetGroupFCMsByID(user_id)
	if len(fcms) > 0 {
		FirebaseMessaging.SendMessage(Models.NotificationRequest{Tokens: fcms, Title: "An Appointment Has Been Accepted", Body: fmt.Sprintf("Your appointment at %s with %s has been accepted", appointmentRequest.DateTime, appointmentRequest.PatientName)})
	}
	SSE.Broadcaster.Broadcast("refresh")

	if appointmentTime.After(time.Now()) {
		// Split the datetime string
		// Split the datetime string
		datetimeParts := strings.Split(appointmentRequest.DateTime, " & ")
		date := datetimeParts[0]
		time := datetimeParts[1]

		// Reformat date from yyyy/MM/dd to dd/MM/yyyy
		dateParts := strings.Split(date, "/")
		if len(dateParts) == 3 {
			// Rearrange from yyyy/MM/dd to dd/MM/yyyy
			date = fmt.Sprintf("%s/%s/%s", dateParts[2], dateParts[1], dateParts[0])
		}

		// Remove "Dr." prefix if it exists
		therapistName := appointmentRequest.TherapistName
		therapistName = strings.TrimPrefix(therapistName, "Dr. ")
		therapistName = strings.TrimPrefix(therapistName, "د. ")
		therapistName = strings.TrimPrefix(therapistName, "Dr.")
		therapistName = strings.TrimPrefix(therapistName, "د.")

		// Convert date to Arabic format (replace Western numbers with Arabic numbers)
		arabicDate := strings.NewReplacer(
			"0", "٠",
			"1", "١",
			"2", "٢",
			"3", "٣",
			"4", "٤",
			"5", "٥",
			"6", "٦",
			"7", "٧",
			"8", "٨",
			"9", "٩",
			"/", "/",
		).Replace(date)

		// Convert time to Arabic format
		arabicTime := strings.NewReplacer(
			"0", "٠",
			"1", "١",
			"2", "٢",
			"3", "٣",
			"4", "٤",
			"5", "٥",
			"6", "٦",
			"7", "٧",
			"8", "٨",
			"9", "٩",
			"AM", "صباحًا",
			"PM", "مساءً",
		).Replace(time)

		message := fmt.Sprintf("🗓️ *APPOINTMENT CONFIRMATION* 🗓️\\n\\n"+
			"Dear Patient,\\n\\n"+
			"Your appointment has been confirmed:\\n"+
			"• *Date:* %s\\n"+
			"• *Time:* %s\\n"+
			"• *Therapist:* Dr. %s\\n\\n"+
			"✅ *تأكيد الموعد* ✅\\n\\n"+
			"عزيزي المريض،\\n\\n"+
			"تم تأكيد موعدك:\\n"+
			"• *التاريخ:* %s\\n"+
			"• *الوقت:* %s\\n"+
			"• *دكتور:* %s\\n\\n"+
			"Please arrive 10 minutes early. If you need to reschedule, kindly contact us 24 hours in advance.\\n"+
			"يرجى الحضور قبل الموعد بـ 10 دقائق. إذا كنت بحاجة إلى إعادة الجدولة، يرجى الاتصال بنا قبل 24 ساعة.",
			date,
			time,
			therapistName,
			arabicDate,
			arabicTime,
			therapistName)

		Whatsapp.SendMessage(appointmentRequest.PhoneNumber, message)
	}
}

func RegisterAppointment(c *gin.Context) {
	var input struct {
		AppointmentID uint                 `json:"appointment_id"`
		TreatmentPlan Models.TreatmentPlan `json:"treatment_plan"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := Models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("paniced")
			tx.Rollback() // Rollback the transaction in case of panic
		}
	}()

	var appointment Models.Appointment

	if err := tx.Model(&Models.Appointment{}).Where("id = ?", input.AppointmentID).First(&appointment).Error; err != nil {
		log.Println(err.Error())
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Super treatment plan not found"})
		return
	}

	if input.TreatmentPlan.ID == 0 {
		// Create a new treatment plan
		if err := tx.Model(&Models.SuperTreatmentPlan{}).Where("id = ?", input.TreatmentPlan.SuperTreatmentPlanID).First(&input.TreatmentPlan.SuperTreatmentPlan).Error; err != nil {
			log.Println(err.Error())
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Super treatment plan not found"})
			return
		}

		input.TreatmentPlan.PatientID = appointment.PatientID
		input.TreatmentPlan.TotalPrice = input.TreatmentPlan.SuperTreatmentPlan.Price * ((100 - input.TreatmentPlan.Discount) / 100)
		input.TreatmentPlan.Remaining = input.TreatmentPlan.SuperTreatmentPlan.SessionsCount

		if err := tx.Create(&input.TreatmentPlan).Error; err != nil {
			log.Println(err.Error())
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create treatment plan"})
			return
		}
		defer func() {
			user_id, err := Token.ExtractTokenID(c)
			if err != nil {
				log.Println(err)
			}
			fcms, _ := Models.GetGroupFCMsByID(user_id)
			if len(fcms) > 0 {
				FirebaseMessaging.SendMessage(Models.NotificationRequest{Tokens: fcms, Title: "A Package Has Been Registered", Body: fmt.Sprintf("%s has registered \"%s\" with a price of: %v", appointment.PatientName, input.TreatmentPlan.SuperTreatmentPlan.Description, input.TreatmentPlan.TotalPrice)})
			}
		}()

	}
	fmt.Println(input.TreatmentPlan.ID)
	fmt.Println(input.AppointmentID)
	if err := tx.Model(&Models.Appointment{}).Where("id = ?", input.AppointmentID).Update("treatment_plan_id", input.TreatmentPlan.ID).Error; err != nil {
		log.Println(err.Error())
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create treatment plan"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}
	SSE.Broadcaster.Broadcast("refresh")
	c.JSON(http.StatusOK, gin.H{"message": "Appointment Registered Successfully"})

}

func RejectAppointment(c *gin.Context) {
	var input struct {
		ID uint `json:"ID"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, err)
		return
	}

	// Start a transaction
	tx := Models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback() // Rollback the transaction in case of panic
		}
	}()

	var appointmentReq Models.AppointmentRequest

	if err := tx.Model(&Models.AppointmentRequest{}).Where("id = ?", input.ID).First(&appointmentReq).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		return
	}

	if err := tx.Delete(&Models.AppointmentRequest{}, "id = ?", input.ID).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}
	user_id, err := Token.ExtractTokenID(c)
	if err != nil {
		log.Println(err)
	}
	fcms, _ := Models.GetGroupFCMsByID(user_id)
	if len(fcms) > 0 {
		FirebaseMessaging.SendMessage(Models.NotificationRequest{Tokens: fcms, Title: "An Appointment Has Been Rejected", Body: fmt.Sprintf("Your appointment at %s with %s has been rejected", appointmentReq.DateTime, appointmentReq.PatientName)})
	}
	SSE.Broadcaster.Broadcast("refresh")
	appointmentTime, err := time.Parse("2006/01/02 & 3:04 PM", appointmentReq.DateTime)

	if appointmentTime.After(time.Now()) {
		message := fmt.Sprintf("❌ *APPOINTMENT REJECTED* ❌\\n\\n" +
			"Dear Patient,\\n\\n" +
			"We're sorry, but your appointment request has been rejected. Please contact the clinic to reschedule or for further information.\\n\\n" +
			"❌ *تم رفض الموعد* ❌\\n\\n" +
			"عزيزي المريض،\\n\\n" +
			"نعتذر، ولكن تم رفض طلب موعدك. يرجى الاتصال بالعيادة لإعادة الجدولة أو للحصول على مزيد من المعلومات.")

		Whatsapp.SendMessage(appointmentReq.PhoneNumber, message)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rejected Successfully"})
}

func MarkAppointmentAsCompleted(c *gin.Context) {
	var input struct {
		ID uint `json:"ID"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, err)
		return
	}

	// Start a transaction
	tx := Models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback() // Rollback the transaction in case of panic
		}
	}()

	var treatmentPlanID uint

	if err := tx.Model(&Models.Appointment{}).Where("id = ?", input.ID).Select("treatment_plan_id").Find(&treatmentPlanID).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	var TreatmentPlan Models.TreatmentPlan

	if err := tx.Model(&Models.TreatmentPlan{}).Where("id = ?", treatmentPlanID).First(&TreatmentPlan).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	TreatmentPlan.Remaining -= 1

	if err := tx.Save(&TreatmentPlan).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	if err := tx.Model(&Models.Appointment{}).Where("id = ?", input.ID).Update("is_completed", true).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked Successfully"})
}

func UnmarkAppointmentAsCompleted(c *gin.Context) {

	var input struct {
		ID uint `json:"ID"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, err)
		return
	}

	// Start a transaction
	tx := Models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback() // Rollback the transaction in case of panic
		}
	}()

	var treatmentPlanID uint

	if err := tx.Model(&Models.Appointment{}).Where("id = ?", input.ID).Select("treatment_plan_id").Find(&treatmentPlanID).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	var TreatmentPlan Models.TreatmentPlan

	if err := tx.Model(&Models.TreatmentPlan{}).Where("id = ?", treatmentPlanID).First(&TreatmentPlan).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	TreatmentPlan.Remaining += 1

	if err := tx.Save(&TreatmentPlan).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	if err := tx.Model(&Models.Appointment{}).Where("id = ?", input.ID).Update("is_completed", false).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked Successfully"})
}

func RemoveAppointmentSendMessage(c *gin.Context) {
	var input struct {
		ID uint `json:"ID"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	// Start a transaction
	tx := Models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback() // Rollback the transaction in case of panic
		}
	}()

	var TimeBlock Models.TimeBlock

	if err := tx.Model(&Models.TimeBlock{}).Where("id = ?", input.ID).Preload("Appointment").First(&TimeBlock).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}
	var Patient Models.Patient
	tx.Model(&Models.Patient{}).Where("id = ?", TimeBlock.Appointment.PatientID).First(&Patient)

	if err := tx.Model(&Models.TimeBlock{}).Delete("id = ?", input.ID).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	if err := tx.Where("time_block_id = ?", input.ID).Delete(&Models.Appointment{}).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted Successfully"})

	if Patient.Phone != "" {
		user_id, err := Token.ExtractTokenID(c)
		if err != nil {
			log.Println(err)
		}
		fcms, _ := Models.GetGroupFCMsByID(user_id)
		if len(fcms) > 0 {
			go FirebaseMessaging.SendMessage(Models.NotificationRequest{Tokens: fcms, Title: "Appointment Cancelled", Body: fmt.Sprintf("Your Appointment With %s, At %s Has Been Cancelled", Patient.Name, TimeBlock.DateTime)})
		}

		appointmentTime, err := time.Parse("2006/01/02 & 3:04 PM", TimeBlock.DateTime)
		if appointmentTime.After(time.Now()) {
			message := fmt.Sprintf("🚫 *APPOINTMENT DELETED* 🚫\\n\\n" +
				"Dear Patient,\\n\\n" +
				"We're sorry, but your appointment has been deleted. Please contact the clinic to reschedule at your earliest convenience.\\n\\n" +
				"🚫 *تم إلغاء الموعد* 🚫\\n\\n" +
				"عزيزي المريض،\\n\\n" +
				"نعتذر، ولكن تم إلغاء موعدك. يرجى الاتصال بالعيادة لإعادة الجدولة في أقرب وقت مناسب لك.")

			go Whatsapp.SendMessage(Patient.Phone, message)
		}
	}

}

func RemovePackage(c *gin.Context) {
	var input struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}
	tx := Models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback() // Rollback the transaction in case of panic
		}
	}()
	var treatmentPlan Models.TreatmentPlan
	if err := tx.Model(&Models.TreatmentPlan{}).Preload("Appointments").Where("id = ?", &input.ID).First(&treatmentPlan).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	for _, appointment := range treatmentPlan.Appointments {
		if err := tx.Delete(&Models.TimeBlock{}, "id = ?", appointment.TimeBlockID).Error; err != nil {
			log.Println(err)
			tx.Rollback()
			c.JSON(http.StatusBadRequest, err)
			c.Abort()
			return
		}
		if err := tx.Delete(&Models.Appointment{}, "id = ?", appointment.ID).Error; err != nil {
			log.Println(err)
			tx.Rollback()
			c.JSON(http.StatusBadRequest, err)
			c.Abort()
			return
		}
	}
	if err := tx.Delete(&Models.TreatmentPlan{}, "id = ?", treatmentPlan.ID).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Package Deleted Successfully",
	})
}

func DeletePatient(c *gin.Context) {
	var input struct {
		PatientID uint `json:"patient_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	// Start a transaction
	tx := Models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback() // Rollback the transaction in case of panic
		}
	}()

	if err := tx.Delete(&Models.Patient{}, "id = ?", input.PatientID).Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Patient Deleted Successfully",
	})
}
