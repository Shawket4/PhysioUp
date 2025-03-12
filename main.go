package main

import (
	"PhysioUp/Models"
	"PhysioUp/Routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	Models.ConnectDataBase()
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://physioup.ddns.net", "http://localhost:3000"}, // Replace with your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true, // Allow cookies)
	},
	))
	Routes.ConfigRoutes(router)
	// go func() {

	// }()
	// router.RunTLS(":5505", "dentex.crt", "dentex_priv.key")
	router.Run(":3005")
}
