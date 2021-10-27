package main

import (
	"os"
	"reloading-api/controllers"
	"reloading-api/middlewares"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.New()
	router.Use(gin.Recovery(), middlewares.CustomizedLogging())
	router.GET("/", controllers.Index)
	getRoutergroup := router.Group("/get", gin.BasicAuth(gin.Accounts{
		os.Getenv("GETGROUPUSER"): os.Getenv("GETGROUPWD"),
	}))
	adminGroup := router.Group("/admin", gin.BasicAuth(gin.Accounts{
		os.Getenv("USERADMIN"): os.Getenv("USERADMINPWD"),
	}))
	getRoutergroup.GET("/by-stocksymbol", controllers.GetStockByName)
	getRoutergroup.GET("/allstocks", controllers.GetAllStocks)
	adminGroup.GET("", controllers.HelloAdmin)
	adminGroup.GET("/fetch", controllers.FetchAndParseStockData)
	router.GET("/trunc", controllers.TruncTable)

	router.Run(":8080")

}
