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
	// Handle treatment plan
	// if input.TreatmentPlan.ID == 0 {
	// 	// Create a new treatment plan
	// 	var superTreatmentPlan Models.SuperTreatmentPlan
	// 	if err := tx.Model(&Models.SuperTreatmentPlan{}).Where("id = ?", input.TreatmentPlan.SuperTreatmentPlanID).First(&superTreatmentPlan).Error; err != nil {
	// 		log.Println(err.Error())
	// 		tx.Rollback()
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Super treatment plan not found"})
	// 		return
	// 	}

	// 	input.TreatmentPlan.PatientID = appointmentRequest.PatientID
	// 	input.TreatmentPlan.TotalPrice = superTreatmentPlan.Price * ((100 - input.TreatmentPlan.Discount) / 100)
	// 	input.TreatmentPlan.Remaining = superTreatmentPlan.SessionsCount
	// 	if err := tx.Create(&input.TreatmentPlan).Error; err != nil {
	// 		log.Println(err.Error())
	// 		tx.Rollback()
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create treatment plan"})
	// 		return
	// 	}
	// }
	// appointment.TreatmentPlanID = input.TreatmentPlan.ID

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

	// Send confirmation message (optional)
	// client := twilio.NewRestClient()
	// params := &api.CreateMessageParams{}
	// parts := strings.Split(input.Extra.DateTime, " & ")
	// params.SetBody(fmt.Sprintf("Your Appointment at: %s is confirmed with Dr. %s\nتم تأكيد الحجز علي الميعاد الساعة %s يوم %s مع دكتور %s", appointmentRequest.DateTime, appointmentRequest.TherapistName, parts[1], parts[0], appointment.TherapistName))
	// params.SetFrom("+15076936009")
	// params.SetTo(appointmentRequest.PhoneNumber)
	// resp, err := client.Api.CreateMessage(params)
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	os.Exit(1)
	// } else {
	// 	if resp.Body != nil {
	// 		fmt.Println(*resp.Body)
	// 	} else {
	// 		fmt.Println(resp.Body)
	// 	}
	// }
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
	Whatsapp.SendMessage(appointmentRequest.PhoneNumber, fmt.Sprintf("Your Appointment At %s With %s Has Been Confirmed", appointmentRequest.DateTime, appointmentRequest.TherapistName))
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
	Whatsapp.SendMessage(appointmentReq.PhoneNumber, "We're sorry. Your appointment has been rejected, please contact the clinic to reschedule")
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

	// if err := Models.DB.Model(&Models.Appointment{}).Delete("id = ?", appointment.ID).Error; err != nil {
	// 	log.Println(err)
	// 	c.JSON(http.StatusBadRequest, err)
	// 	c.Abort()
	// 	return
	// }

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted Successfully"})
	if Patient.Phone != "" {
		Whatsapp.SendMessage(Patient.Phone, "We're sorry. Your appointment has been deleted, please contact the clinic to reschedule")
		user_id, err := Token.ExtractTokenID(c)
		if err != nil {
			log.Println(err)
		}
		fcms, _ := Models.GetGroupFCMsByID(user_id)
		if len(fcms) > 0 {
			FirebaseMessaging.SendMessage(Models.NotificationRequest{Tokens: fcms, Title: "Appointment Cancelled", Body: fmt.Sprintf("Your Appointment With %s, At %s Has Been Cancelled", Patient.Name, TimeBlock.DateTime)})
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
