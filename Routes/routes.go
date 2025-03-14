package Routes

import (
	"PhysioUp/Controllers"
	"PhysioUp/SSE"
	"PhysioUp/Whatsapp"

	"PhysioUp/Middleware"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func ConfigRoutes(router *gin.Engine) {
	router.Use(gzip.Gzip(gzip.BestSpeed))
	authorized := router.Group("/api/protected")
	authorized.Use(Middleware.JwtAuthMiddleware())
	authorized.GET("/user", Controllers.CurrentUser)

	public := router.Group("/api")
	public.Use()
	// router.GET("/stream", SSE.HeadersMiddleware(), SSE.Stream.SSEConnMiddleware(), SSE.RequestSSE)
	public.POST("/login", Controllers.Login)
	public.POST("/SaveFcmToken", Controllers.SaveFcmToken)
	public.POST("/register", Controllers.Register)
	public.POST("/RequestAppointment", Controllers.RequestAppointment)
	public.POST("/GetPatientIdByPhone", Controllers.GetPatientIdByPhone)
	public.POST("/FetchAppointmentsByPatientID", Controllers.FetchAppointmentsByPatientID)
	public.POST("/FetchFutureAppointments", Controllers.FetchFutureAppointments)
	public.POST("/VerifyAppointmentRequestPhoneNo", Controllers.VerifyAppointmentRequestPhoneNo)
	authorized.GET("/CheckWhatsAppLogin", Whatsapp.CheckLogin)
	authorized.GET("/GetWhatsAppQRCode", Whatsapp.GetQRCode)
	authorized.GET("/FetchRequestedAppointments", Controllers.FetchRequestedAppointments)
	authorized.GET("/FetchUnassignedAppointments", Controllers.FetchUnassignedAppointments)
	authorized.POST("/FetchPatientCurrentPackage", Controllers.FetchPatientCurrentPackage)
	authorized.POST("/FetchPatientPackages", Controllers.FetchPatientPackages)
	authorized.POST("/ExportSalesTable", Controllers.ExportSalesTable)
	authorized.POST("/FetchPackageAppointments", Controllers.FetchPackageAppointments)
	authorized.POST("/MarkPackageAsPaid", Controllers.MarkPackageAsPaid)
	authorized.POST("/UnMarkPackageAsPaid", Controllers.UnMarkPackageAsPaid)
	authorized.GET("/RequestSSE", SSE.RequestSSE)
	authorized.POST("/AcceptAppointment", Controllers.AcceptAppointment)
	authorized.POST("/RegisterAppointment", Controllers.RegisterAppointment)
	authorized.POST("/RejectAppointment", Controllers.RejectAppointment)
	authorized.POST("/MarkAppointmentAsCompleted", Controllers.MarkAppointmentAsCompleted)
	authorized.POST("/UnmarkAppointmentAsCompleted", Controllers.UnmarkAppointmentAsCompleted)
	authorized.POST("/RemoveAppointment", Controllers.RemoveAppointment)
	authorized.POST("/RemovePackage", Controllers.RemovePackage)
	public.POST("/RegisterTherapist", Controllers.RegisterTherapist)
	authorized.POST("/GetTherapistSchedule", Controllers.GetTherapistSchedule)
	authorized.POST("/AddTherapistTimeBlocks", Controllers.AddTherapistTimeBlocks)
	authorized.GET("/FetchPatients", Controllers.FetchPatients)
	authorized.GET("/GetTherapists", Controllers.GetTherapists)
	public.GET("/GetTherapistsTrimmed", Controllers.GetTherapistsTrimmed)
	authorized.POST("/FetchPatientFilesURLs", Controllers.FetchPatientFilesURLs)
	authorized.POST("/FetchReferralPackages", Controllers.FetchReferralPackages)
	authorized.POST("/ExportReferredPackagesExcel", Controllers.ExportReferredPackagesExcel)
	authorized.POST("/UploadPatientRecord", Controllers.UploadPatientRecord)
	authorized.POST("/DeletePatientRecord", Controllers.DeletePatientRecord)
	authorized.POST("/UpdatePatient", Controllers.UpdatePatient)
	authorized.POST("/CreatePatient", Controllers.CreatePatient)
	authorized.GET("/FetchReferrals", Controllers.FetchReferrals)
	authorized.POST("/AddReferral", Controllers.AddReferral)
	authorized.POST("/EditReferral", Controllers.EditReferral)
	authorized.POST("/DeleteReferral", Controllers.DeleteReferral)
	public.GET("/FetchSuperTreatments", Controllers.FetchSuperTreatments)
	authorized.POST("/AddSuperTreatment", Controllers.AddSuperTreatment)
	authorized.POST("/EditSuperTreatment", Controllers.EditSuperTreatment)
	authorized.POST("/DeleteSuperTreatment", Controllers.DeleteSuperTreatment)
	authorized.POST("/SetPackageReferral", Controllers.SetPackageReferral)
	authorized.POST("/DeletePatient", Controllers.DeletePatient)
	authorized.Static("/PatientRecords", "./PatientRecords")
	router.Static("/Web", "./Static")
	router.Static("/Welcome", "./Welcome")
}
