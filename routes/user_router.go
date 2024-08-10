package routes

import (
	"github.com/yonatan-tize/go-auth/controllers"
	"github.com/gin-gonic/gin"
	"github.com/yonatan-tize/go-auth/middleware"
)

func UserRoutes(incomingRoutes *gin.Engine){

	incomingRoutes.Use(middleware.Authenticate())
	incomingRoutes.GET("/users", controllers.GetUsers())
	incomingRoutes.GET("/users/:user_id", controllers.GetUser())
}