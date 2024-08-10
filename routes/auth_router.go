package routes

import (
	"github.com/yonatan-tize/go-auth/controllers"
	"github.com/gin-gonic/gin"
)


func AuthRoutes(incomingRoutes *gin.Engine){
	incomingRoutes.POST("users/signup", controllers.SignUp())
	incomingRoutes.POST("users/signup", controllers.Login())

}