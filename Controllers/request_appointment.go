package Controllers

import (
	"PhysioUp/Models"
	"PhysioUp/SSE"
	"PhysioUp/Utils/Token"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RequestAppointment(c *gin.Context) {
	var input Models.AppointmentRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := Models.DB.Begin()

	// Check if the patient already has an appointment on the same day

	var therapist Models.Therapist
	if err := tx.Model(&Models.Therapist{}).Where("id = ?", input.TherapistID).Preload("Schedule.TimeBlocks").First(&therapist).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user_id, _ := Token.ExtractTokenID(c)
	var user Models.User
	if user_id != 0 {
		user, _ = Models.GetUserByID(user_id)
	}

	if user.Permission < 2 {

		layoutWithLeadingZero := "2006/01/02 & 03:04 PM"
		layoutWithoutLeadingZero := "2006/01/02 & 3:04 PM"

		// Try parsing with both layouts
		var parsedTime time.Time
		var err error

		parsedTime, err = time.Parse(layoutWithLeadingZero, input.DateTime)
		if err != nil {
			// If the first layout fails, try the second layout
			parsedTime, err = time.Parse(layoutWithoutLeadingZero, input.DateTime)
			if err != nil {
				fmt.Println("Error parsing date:", err)
				return
			}
		}

		currentTime := time.Now()

		// Calculate the difference between the parsed time and the current time
		timeDifference := parsedTime.Sub(currentTime)

		// Define two weeks duration
		twoWeeks := 14 * 24 * time.Hour

		if timeDifference > twoWeeks {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Time Block Not Allowed, Can't Book More Than 14 days ahead"})
			return
		}
	}

	// Check if the time block is already booked
	for _, timeblock := range therapist.Schedule.TimeBlocks {
		if timeblock.DateTime == input.DateTime {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Time Block already booked"})
			return
		}
	}

	// Set therapist name in input
	input.TherapistName = therapist.Name
	input.TherapistID = therapist.ID

	if input.PatientID == 0 {
		// Begin Transaction

		// If an error occurs, rollback the transaction
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong, transaction rolled back"})
			}
		}()

		if !strings.HasPrefix(input.PhoneNumber, "+") {
			input.PhoneNumber = "+2" + input.PhoneNumber
		}

		var patient Models.Patient

		if err := tx.Model(&Models.Patient{}).Where("phone = ?", input.PhoneNumber).First(&patient).Error; true {
			if errors.Is(err, gorm.ErrRecordNotFound) && !input.IsExisting {
				patient.Name = input.PatientName
				patient.Phone = input.PhoneNumber
				patient.GenerateOTPToken(6)
				patient.IsVerified = true
				if err := tx.Create(&patient).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusBadRequest, gin.H{"error": "Couldn't Create Patient"})
					return
				}
			} else if errors.Is(err, gorm.ErrRecordNotFound) && input.IsExisting {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Phone Number Not Registered, Try Registering As a New Patient"})
				return
			} else if err == nil && !input.IsExisting {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Phone Already Registered, Try Requesting As an Existing Patient"})
				return
			}
		}

		input.PatientID = patient.ID
		var existingAppointmentRequests []Models.AppointmentRequest
		var existingAppointments []Models.Appointment

		if err := tx.Model(&Models.AppointmentRequest{}).
			Where("patient_id = ? AND DATE(date_time) = ?", input.PatientID, input.DateTime).
			Find(&existingAppointmentRequests).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to check existing appointments"})
			return
		}

		if err := tx.Model(&Models.Appointment{}).
			Where("patient_id = ? AND DATE(date_time) = ?", input.PatientID, input.DateTime).
			Find(&existingAppointments).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to check existing appointments"})
			return
		}

		if len(existingAppointmentRequests) > 0 || len(existingAppointments) > 0 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Patient can only book one appointment per day"})
			return
		}
		input.PatientName = patient.Name
		input.PhoneNumber = patient.Phone
	} else {
		var patient Models.Patient
		if err := tx.Model(&Models.Patient{}).Where("id = ?", input.PatientID).First(&patient).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Couldn't Create Patient"})
			return
		}
		input.PatientName = patient.Name
		input.PhoneNumber = patient.Phone
	}

	// Save the appointment request
	if err := tx.Save(&input).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Commit the transaction if everything is successful
	tx.Commit()
	SSE.Broadcaster.Broadcast("refresh")
	c.SetCookie("patient_id", fmt.Sprintf("%d", input.PatientID), 3600*24*14, "/", "/", false, false)
	c.SetCookie("phone_number", fmt.Sprintf("%s", input.PhoneNumber), 3600*24*14, "/", "/", false, false)
	c.SetCookie("patient_name", fmt.Sprintf("%s", input.PatientName), 3600*24*14, "/", "/", false, false)
	c.JSON(http.StatusOK, gin.H{
		"message":        "Requested Successfully",
		"appointment_id": input.ID,
		"patient_id":     input.PatientID,
	})
}

func FetchRequestedAppointments(c *gin.Context) {
	var output []Models.AppointmentRequest
	if err := Models.DB.Model(&Models.AppointmentRequest{}).Joins("JOIN patients ON patients.id = appointment_requests.patient_id").
		Where("patients.is_verified = ?", true).Find(&output).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, output)
}

func FetchUnassignedAppointments(c *gin.Context) {
	var output []Models.Appointment
	if err := Models.DB.Model(&Models.Appointment{}).
		Where("treatment_plan_id IS null").Find(&output).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(output)
	c.JSON(http.StatusOK, output)
}

func FetchPatientCurrentPackage(c *gin.Context) {
	var input struct {
		PatientID uint `json:"patient_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var Package Models.TreatmentPlan
	if err := Models.DB.Model(&Models.TreatmentPlan{}).Where("patient_id = ?", input.PatientID).Last(&Package).Error; err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	if err := Models.DB.Model(&Models.SuperTreatmentPlan{}).Where("id = ?", Package.SuperTreatmentPlanID).Find(&Package.SuperTreatmentPlan).Error; err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, Package)
}

func FetchPatientPackages(c *gin.Context) {
	var input struct {
		PatientID uint `json:"patient_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var Packages []Models.TreatmentPlan
	if err := Models.DB.Model(&Models.TreatmentPlan{}).Where("patient_id = ?", input.PatientID).Find(&Packages).Error; err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}

	for index := range Packages {
		if err := Models.DB.Model(&Models.SuperTreatmentPlan{}).Where("id = ?", Packages[index].SuperTreatmentPlanID).Find(&Packages[index].SuperTreatmentPlan).Error; err != nil {
			c.JSON(http.StatusOK, nil)
			return
		}
	}

	c.JSON(http.StatusOK, Packages)
}

func MarkPackageAsPaid(c *gin.Context) {
	var input struct {
		PackageID     uint   `json:"package_id"`
		PaymentMethod string `json:"payment_method"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	if err := Models.DB.Model(&Models.TreatmentPlan{}).Where("id = ?", input.PackageID).Update("is_paid", true).Update("payment_method", input.PaymentMethod).Error; err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Marked Successfully"})
}

func UnMarkPackageAsPaid(c *gin.Context) {

	var input struct {
		PackageID uint `json:"package_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	if err := Models.DB.Model(&Models.TreatmentPlan{}).Where("id = ?", input.PackageID).Update("is_paid", false).Update("payment_method", "").Error; err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Marked Successfully"})
}

func FetchPackageAppointments(c *gin.Context) {
	var input struct {
		PackageID uint `json:"package_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var Appointments []Models.Appointment
	if err := Models.DB.Model(&Models.Appointment{}).Where("treatment_plan_id = ?", input.PackageID).Find(&Appointments).Error; err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}

	c.JSON(http.StatusOK, Appointments)
}

func VerifyAppointmentRequestPhoneNo(c *gin.Context) {
	var input struct {
		ID  uint   `json:"ID"`
		OTP string `json:"otp"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var patientID uint

	if err := Models.DB.Model(&Models.AppointmentRequest{}).Where("id = ?", input.ID).Select("patient_id").First(&patientID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var patientOTP string
	var patientPhone string
	if err := Models.DB.Model(&Models.Patient{}).Where("id = ?", patientID).Select("phone").First(&patientPhone).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := Models.DB.Model(&Models.Patient{}).Where("id = ?", patientID).Select("otp").First(&patientOTP).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(input.ID)
	fmt.Println(patientOTP)
	if patientOTP == input.OTP {
		if err := Models.DB.Model(&Models.Patient{}).Where("id = ?", patientID).Update("is_verified", true).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		SSE.Broadcaster.Broadcast("refresh")
		// client := twilio.NewRestClient()

		// params := &api.CreateMessageParams{}
		// params.SetBody(fmt.Sprintf("Your ID for future appointments is: %v", patientID))
		// params.SetFrom("+15076936009")
		// params.SetTo(patientPhone)

		// resp, err := client.Api.CreateMessage(params)
		// if err != nil {
		// 	fmt.Println(err.Error())
		// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send OTP"})
		// 	return
		// } else {
		// 	if resp.Body != nil {
		// 		fmt.Println(*resp.Body)
		// 	} else {
		// 		fmt.Println(resp.Body)
		// 	}
		// }
		c.JSON(http.StatusOK, gin.H{"message": "Phone Number Confirmed"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.New("Incorrect OTP")})
		return
	}
}

func FetchAppointmentsByPatientID(c *gin.Context) {
	var input struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For Appointments: Select only the required fields
	type AppointmentResponse struct {
		ID            uint   `json:"id"`
		DateTime      string `json:"date_time"`
		TherapistName string `json:"therapist_name"`
		IsCompleted   bool   `json:"is_completed"`
	}

	var appointmentResponses []AppointmentResponse
	if err := Models.DB.Model(&Models.Appointment{}).
		Select("id, date_time, therapist_name, is_completed").
		Where("patient_id = ?", input.ID).
		Find(&appointmentResponses).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For AppointmentRequests: Select only the required fields
	type RequestResponse struct {
		ID            uint   `json:"id"`
		DateTime      string `json:"date_time"`
		TherapistName string `json:"therapist_name"`
	}

	var requestResponses []RequestResponse
	if err := Models.DB.Model(&Models.AppointmentRequest{}).
		Select("id, date_time, therapist_name").
		Where("patient_id = ?", input.ID).
		Find(&requestResponses).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"appointments": appointmentResponses,
		"requests":     requestResponses,
	})
}

func GetPatientIdByPhone(c *gin.Context) {
	var input struct {
		PhoneNumber string `json:"phone_number"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !strings.HasPrefix(input.PhoneNumber, "+") {
		input.PhoneNumber = "+2" + input.PhoneNumber
	}

	var patient Models.Patient
	if err := Models.DB.Where("phone = ?", input.PhoneNumber).First(&patient).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No patient found with this phone number"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"patient_id": patient.ID})
}
