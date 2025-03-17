package Routes

import (
	"PhysioUp/Controllers"
	"PhysioUp/Middleware"
	"PhysioUp/SSE"
	"PhysioUp/Whatsapp"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func ConfigRoutes(router *gin.Engine) {
	// Gzip Compression
	router.Use(gzip.Gzip(gzip.BestSpeed))

	// Public routes
	public := router.Group("/api")
	{
		public.POST("/login", Controllers.Login)
		public.POST("/register", Controllers.Register)
		public.POST("/register/ClinicGroup", Controllers.RegisterClinicGroup)
		public.POST("/SaveFcmToken", Controllers.SaveFcmToken)
		public.POST("/RequestAppointment", Controllers.RequestAppointment)
		public.POST("/GetPatientIdByPhone", Controllers.GetPatientIdByPhone)
		public.POST("/FetchAppointmentsByPatientID", Controllers.FetchAppointmentsByPatientID)
		public.POST("/FetchFutureAppointments", Controllers.FetchFutureAppointments)
		public.POST("/VerifyAppointmentRequestPhoneNo", Controllers.VerifyAppointmentRequestPhoneNo)
		public.GET("/GetTherapistsTrimmed", Controllers.GetTherapistsTrimmed)
	}

	// Authorized routes
	authorized := router.Group("/api/protected")
	authorized.Use(Middleware.JwtAuthMiddleware())
	authorized.Use(Middleware.SetClinicGroup())
	{

		// User-related routes
		authorized.GET("/user", Controllers.CurrentUser)

		// Appointment-related routes
		authorized.GET("/FetchRequestedAppointments", Controllers.FetchRequestedAppointments)
		authorized.GET("/FetchUnassignedAppointments", Controllers.FetchUnassignedAppointments)
		authorized.POST("/AcceptAppointment", Controllers.AcceptAppointment)
		authorized.POST("/RegisterAppointment", Controllers.RegisterAppointment)
		authorized.POST("/RejectAppointment", Controllers.RejectAppointment)
		authorized.POST("/MarkAppointmentAsCompleted", Controllers.MarkAppointmentAsCompleted)
		authorized.POST("/UnmarkAppointmentAsCompleted", Controllers.UnmarkAppointmentAsCompleted)
		authorized.POST("/RemoveAppointmentSendMessage", Controllers.RemoveAppointmentSendMessage)

		// Package-related routes
		authorized.POST("/FetchPatientCurrentPackage", Controllers.FetchPatientCurrentPackage)
		authorized.POST("/FetchPatientPackages", Controllers.FetchPatientPackages)
		authorized.POST("/FetchPackageAppointments", Controllers.FetchPackageAppointments)
		authorized.POST("/MarkPackageAsPaid", Controllers.MarkPackageAsPaid)
		authorized.POST("/UnMarkPackageAsPaid", Controllers.UnMarkPackageAsPaid)
		authorized.POST("/RemovePackage", Controllers.RemovePackage)
		authorized.POST("/SetPackageReferral", Controllers.SetPackageReferral)

		// Therapist-related routes
		authorized.POST("/RegisterTherapist", Controllers.RegisterTherapist)
		authorized.POST("/DeleteTherapist", Controllers.DeleteTherapist)
		authorized.POST("/GetTherapistSchedule", Controllers.GetTherapistSchedule)
		authorized.POST("/AddTherapistTimeBlocks", Controllers.AddTherapistTimeBlocks)
		authorized.GET("/GetTherapists", Controllers.GetTherapists)

		// Patient-related routes
		authorized.GET("/FetchPatients", Controllers.FetchPatients)
		authorized.POST("/FetchPatientFilesURLs", Controllers.FetchPatientFilesURLs)
		authorized.POST("/UploadPatientRecord", Controllers.UploadPatientRecord)
		authorized.POST("/DeletePatientRecord", Controllers.DeletePatientRecord)
		authorized.POST("/UpdatePatient", Controllers.UpdatePatient)
		authorized.POST("/CreatePatient", Controllers.CreatePatient)
		authorized.POST("/DeletePatient", Controllers.DeletePatient)

		// Referral-related routes
		authorized.GET("/FetchReferrals", Controllers.FetchReferrals)
		authorized.POST("/AddReferral", Controllers.AddReferral)
		authorized.POST("/EditReferral", Controllers.EditReferral)
		authorized.POST("/DeleteReferral", Controllers.DeleteReferral)
		authorized.POST("/FetchReferralPackages", Controllers.FetchReferralPackages)
		authorized.POST("/ExportReferredPackagesExcel", Controllers.ExportReferredPackagesExcel)

		// Super Treatment-related routes
		authorized.GET("/FetchSuperTreatments", Controllers.FetchSuperTreatments)
		authorized.POST("/AddSuperTreatment", Controllers.AddSuperTreatment)
		authorized.POST("/EditSuperTreatment", Controllers.EditSuperTreatment)
		authorized.POST("/DeleteSuperTreatment", Controllers.DeleteSuperTreatment)
		// WhatsApp-related routes
		authorized.GET("/CheckWhatsAppLogin", Whatsapp.CheckLogin)
		authorized.GET("/GetWhatsAppQRCode", Whatsapp.GetQRCode)

		// SSE (Server-Sent Events) route
		authorized.GET("/RequestSSE", SSE.RequestSSE)

		// Export-related routes
		authorized.POST("/ExportSalesTable", Controllers.ExportSalesTable)
	}

	// Static file serving
	authorized.Static("/PatientRecords", "./PatientRecords")
	router.Static("/Web", "./Static")
	router.Static("/Welcome", "./Welcome")
}
