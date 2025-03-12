package Controllers

import (
	"PhysioUp/Models"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/gin-gonic/gin"
)

func DaysBetweenDates(Date1, Date2 string) int {
	// Convert string to time
	t1, _ := time.Parse("2006-01-02", Date1)
	t2, _ := time.Parse("2006-01-02", Date2)
	// Calculate days between dates
	days := t2.Sub(t1).Hours() / 24
	return int(days)
}

func ExportSalesTable(c *gin.Context) {
	var input struct {
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	var TreatmentPlans []Models.TreatmentPlan

	if input.DateFrom != "" && input.DateTo != "" {
		// Days := DaysBetweenDates(input.DateFrom, input.DateTo)
		if err := Models.DB.Model(&Models.TreatmentPlan{}).
			Where("date BETWEEN ? AND ?", input.DateFrom, input.DateTo).
			Find(&TreatmentPlans).Error; err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

	} else {
		if err := Models.DB.Model(&Models.TreatmentPlan{}).Find(&TreatmentPlans).Error; err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}
	}

	for index := range TreatmentPlans {
		if TreatmentPlans[index].ReferralID != nil {
			Models.DB.Model(&Models.Referral{}).Where("id = ?", TreatmentPlans[index].ReferralID).First(&TreatmentPlans[index].Referral)
		}
	}

	headers := map[string]string{
		"A1": "Date",
		"B1": "Revenue",
		"C1": "Referral",
		"D1": "Payment Method",
		"E1": "Paid",
	}
	file := excelize.NewFile()
	sheet := "Packages"
	file.NewSheet(sheet)
	file.DeleteSheet("Sheet1")
	for k, v := range headers {
		file.SetCellValue(sheet, k, v)
	}

	for i := 0; i < len(TreatmentPlans); i++ {
		appendRowSales(sheet, file, i, TreatmentPlans)
	}
	var filename string = fmt.Sprintf("./Sales.xlsx")
	if err := file.SaveAs(filename); err != nil {
		log.Println(err)
	}
	// c.Context().SetContentType("multipart/form-data")
	// return c.Response().SendFile("./tasks.xlsx")
	c.File(filename)
}

func appendRowSales(sheet string, file *excelize.File, index int, rows []Models.TreatmentPlan) (fileWriter *excelize.File) {
	rowCount := index + 2
	file.SetCellValue(sheet, fmt.Sprintf("A%v", rowCount), rows[index].Date)
	file.SetCellValue(sheet, fmt.Sprintf("B%v", rowCount), rows[index].TotalPrice)
	file.SetCellValue(sheet, fmt.Sprintf("C%v", rowCount), rows[index].Referral.Name)
	file.SetCellValue(sheet, fmt.Sprintf("D%v", rowCount), rows[index].PaymentMethod)
	file.SetCellValue(sheet, fmt.Sprintf("E%v", rowCount), rows[index].IsPaid)
	return file

}

func ExportReferredPackagesExcel(c *gin.Context) {
	var input struct {
		ReferralID uint `json:"referral_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var Referral Models.Referral

	if err := Models.DB.Model(&Models.Referral{}).Where("id = ?", input.ReferralID).Find(&Referral).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var TreatmentPlans []Models.TreatmentPlan
	if err := Models.DB.Model(&Models.TreatmentPlan{}).Where("referral_id = ?", input.ReferralID).Find(&TreatmentPlans).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for index := range TreatmentPlans {
		if err := Models.DB.Model(&Models.SuperTreatmentPlan{}).Where("id = ?", TreatmentPlans[index].SuperTreatmentPlanID).Find(&TreatmentPlans[index].SuperTreatmentPlan).Error; err != nil {
			c.JSON(http.StatusOK, nil)
			return
		}
	}

	headers := map[string]string{
		"A1": "Date",
		"B1": "Description",
		"C1": "Price",
		"D1": "Cashback",
	}

	file := excelize.NewFile()
	sheet := "Referrals"
	file.NewSheet(sheet)
	file.DeleteSheet("Sheet1")
	for k, v := range headers {
		file.SetCellValue(sheet, k, v)
	}

	for i := 0; i < len(TreatmentPlans); i++ {
		appendRowReferral(sheet, file, i, TreatmentPlans, Referral.CashbackPercentage)
	}
	var filename string = fmt.Sprintf("./Referrals.xlsx")
	if err := file.SaveAs(filename); err != nil {
		log.Println(err)
	}
	// c.Context().SetContentType("multipart/form-data")
	// return c.Response().SendFile("./tasks.xlsx")
	c.File(filename)

}

func appendRowReferral(sheet string, file *excelize.File, index int, rows []Models.TreatmentPlan, percentage float64) (fileWriter *excelize.File) {
	rowCount := index + 2
	file.SetCellValue(sheet, fmt.Sprintf("A%v", rowCount), rows[index].Date)
	file.SetCellValue(sheet, fmt.Sprintf("B%v", rowCount), rows[index].SuperTreatmentPlan.Description)
	file.SetCellValue(sheet, fmt.Sprintf("C%v", rowCount), rows[index].TotalPrice)
	file.SetCellValue(sheet, fmt.Sprintf("D%v", rowCount), percentage/100*rows[index].TotalPrice)
	return file

}
