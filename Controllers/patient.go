package Controllers

import (
	"PhysioUp/Models"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func FetchFutureAppointments(c *gin.Context) {
	var input struct {
		PatientID uint `json:"patient_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var appointments []Models.Appointment
	if err := Models.DB.Model(&Models.Appointment{}).Where("patient_id = ?", input.PatientID).Find(&appointments).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, appointments)
}

func FetchPatients(c *gin.Context) {
	db := getScopedDB(c)
	var patients []Models.Patient
	if err := db.Model(&Models.Patient{}).Preload("History").Find(&patients).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, patients)
}

func FetchPatientFilesURLs(c *gin.Context) {
	type FileInfo struct {
		Name string  `json:"name"`
		Size float64 `json:"size"`
	}

	var FileUrls []FileInfo
	var input struct {
		ID uint `json:"id"`
	}

	// Bind JSON input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Read directory entries
	entries, err := os.ReadDir(fmt.Sprintf("./PatientRecords/%v/", input.ID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Iterate over directory entries
	for _, entry := range entries {
		// Get file info
		fileInfo, err := entry.Info()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if fileInfo.Name() != ".DS_Store" {

			// Append file name and size to the response
			FileUrls = append(FileUrls, FileInfo{
				Name: entry.Name(),
				Size: float64(fileInfo.Size()), // Convert size to float64
			})
		}
	}

	// Return the list of file names and sizes
	c.JSON(http.StatusOK, FileUrls)
}

func UploadPatientRecord(c *gin.Context) {
	// Parse the multipart form, with a max file size of 10MB
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to parse form"})
		return
	}

	// Retrieve the patient ID from the form data
	patientID := c.PostForm("id")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Patient ID is required"})
		return
	}

	// Create the directory if it doesn't exist
	patientDir := fmt.Sprintf("./PatientRecords/%s/", patientID)
	if err := os.MkdirAll(patientDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create patient directory"})
		return
	}

	// Retrieve the files from the form data
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve files from form data"})
		return
	}

	files := form.File["files"] // "files" is the key used in the FormData
	for _, file := range files {
		// Create the file in the patient's directory
		filePath := fmt.Sprintf("%s%s", patientDir, file.Filename)
		out, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create the file"})
			return
		}
		defer out.Close()

		// Copy the uploaded file's content to the newly created file
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to open the file"})
			return
		}
		defer src.Close()

		if _, err := io.Copy(out, src); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to save the file"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Files uploaded successfully"})
}

func DeletePatientRecord(c *gin.Context) {
	var input struct {
		ID       uint   `json:"id"`
		FileName string `json:"file_name"`
	}

	// Bind JSON input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Construct the file path
	filePath := fmt.Sprintf("./PatientRecords/%v/%s", input.ID, input.FileName)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Delete the file
	err := os.Remove(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file: " + err.Error()})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

func UpdatePatient(c *gin.Context) {
	var input struct {
		ID        uint    `json:"id"`
		Name      string  `json:"name"`
		Phone     string  `json:"phone"`
		Gender    string  `json:"gender"`
		Age       int     `json:"age"`
		Weight    float64 `json:"weight"`
		Height    float64 `json:"height"`
		Diagnosis string  `json:"diagnosis"`
		Notes     string  `json:"notes"`
	}
	// Bind JSON input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	var patient Models.Patient
	if err := Models.DB.Model(&Models.Patient{}).Where("id = ?", input.ID).Find(&patient).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	if !strings.HasPrefix(input.Phone, "+") {
		input.Phone = "+2" + input.Phone
	}

	patient.Name = input.Name
	patient.Phone = input.Phone
	patient.Gender = input.Gender
	patient.Age = input.Age
	patient.Weight = input.Weight
	patient.Height = input.Height
	patient.Diagnosis = input.Diagnosis
	patient.Notes = input.Notes

	if err := Models.DB.Save(&patient).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Patient updated successfully"})
}

func CreatePatient(c *gin.Context) {
	var input Models.Patient

	// Bind JSON input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	if !strings.HasPrefix(input.Phone, "+") {
		input.Phone = "+2" + input.Phone
	}
	input.IsVerified = true

	client_group_id, exists := c.Get("clinicGroupID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: Client Group Not Set"})
		return
	}

	input.ClinicGroupID = client_group_id.(uint)

	if err := Models.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Patient updated successfully"})
}
