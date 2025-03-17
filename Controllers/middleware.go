package Controllers

import (
	"PhysioUp/Models"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getScopedDB(c *gin.Context) *gorm.DB {
	db, exists := c.Get("db")
	if !exists {
		fmt.Println("dne")
		return Models.DB // Return the default DB if scoped DB doesn't exist
	}

	// Check if db is a function that requires a table name
	dbFunc, ok := db.(func(string) *gorm.DB)
	if ok {
		// Use the model name from the controller context
		// This would need to be set in each controller
		modelName, exists := c.Get("modelName")
		if exists {
			tableName, ok := modelName.(string)
			if ok {
				return dbFunc(tableName)
			}
		}
		// Default to appointment_requests if no model name is set
	}

	// Try the old way as fallback
	return dbFunc("")
}
