package controllers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/yonatan-tize/go-auth/database"
	"github.com/yonatan-tize/go-auth/helpers"
	"github.com/yonatan-tize/go-auth/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
) 

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")
var validate = validator.New()

func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(bytes)
}

func VerifyPassword(userPassword string, foundPassword string) (bool, string){
	err := bcrypt.CompareHashAndPassword([]byte(foundPassword), []byte(userPassword))
	check := true
	message := ""

	if err != nil{
		message = "password not correct"
		check = false
	}
	return check, message
}

func SignUp() gin.HandlerFunc{
	return func(c *gin.Context){
		ctx, close := context.WithTimeout(context.Background(), 1000 * time.Second)
		defer close()
		var user models.User

		if err := c.ShouldBindJSON(&user); err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return 
		}

		validationErr := validate.Struct(user)
		if validationErr != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return 
		}
		
		password := HashPassword(*user.Password)
		user.Password = &password

		count, err := userCollection.CountDocuments(ctx, bson.M{"email":user.Email})
		if err != nil{
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error() })
			return 
		}
		
		if count > 0{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "this email already exists"})
			return
		}

		user.Created_at, _ = time.Parse(time.RFC3339,time.Now().Format(time.RFC3339))
		user.Updated_at, _ = time.Parse(time.RFC3339,time.Now().Format(time.RFC3339))
		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()
		token, refreshToken, _ := helpers.GenerateAllTokens(*user.Email, *user.First_name, *user.Last_name, *user.User_type, user.User_id)
		user.Token = &token
		user.Refresh_token = &refreshToken

		resultInsertionNumber, insertErr := userCollection.InsertOne(ctx, user)

		if insertErr != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User item was not created"})
			return 
		}

		c.JSON(http.StatusOK, resultInsertionNumber)
	}
}

func Login() gin.HandlerFunc{
	return func(c *gin.Context){
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		
		var user models.User
		var foundUser models.User

		if err := c.ShouldBindJSON(&user); err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := userCollection.FindOne(ctx, bson.M{"email":user.Email}).Decode(&foundUser)
		if err != nil{
			if err == mongo.ErrNoDocuments{
				c.JSON(http.StatusUnauthorized, gin.H{"error": "email is incorrect"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get data from database"})
			return
		}
		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
		
		if !passwordIsValid{
			c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
			return 
		}

		token, refreshToken, err := helpers.GenerateAllTokens(*foundUser.Email, *foundUser.First_name, *foundUser.Last_name, *foundUser.User_type, foundUser.User_id)

		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
			return
		}
		helpers.UpdateAllTokens(token, refreshToken, foundUser.User_id)
		err = userCollection.FindOne(ctx, bson.M{"user_id": foundUser.User_id}).Decode(&foundUser)
		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, foundUser)
	}
}

func GetUsers()gin.HandlerFunc{
	return func(c *gin.Context){
		if err:=helpers.CheckUserType(c, "ADMIN"); err != nil{
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		recordPerPage, err := strconv.Atoi(c.Query("recordePerPage"))
		if err != nil || recordPerPage < 1{
			recordPerPage = 10
		}

		page, err1 := strconv.Atoi(c.Query("page"))
		if err1 != nil{
			page = 1
		}

		startIndex := (page - 1) * recordPerPage
		startIndex, err = strconv.Atoi(c.Query("startIndex"))

		matchStage := bson.D{{Key: "$match", Value: bson.D{{}}}}
		groupStage := bson.D{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "null"},
				{Key: "total_count", Value: bson.D{{Key: "$sum", Value: 1}}},
				{Key: "data", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}},
			}},
		}
		projectStage := bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "total_count", Value: 1},
				{Key: "user_items", Value: bson.D{
					{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}},
				}},
			}},
		}

		result, err := userCollection.Aggregate(ctx, mongo.Pipeline{
			matchStage, groupStage, projectStage,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing user items"})
			return
		}
		defer result.Close(ctx)

		var allUsers []bson.M
		if err = result.All(ctx, &allUsers); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while decoding user items"})
			return
		}

		if len(allUsers) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "No users found"})
			return
		}
		c.JSON(http.StatusOK, allUsers[0])
	}
}

func GetUser() gin.HandlerFunc{
	return func(c *gin.Context){
		userId := c.Param("user_id")

		if err := helpers.MatchUserTypeToUid(c, userId); err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return 
		}
		ctx, cancel := context.WithTimeout(context.Background(), 100 * time.Second)
		defer cancel()
		var user models.User
		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		if err != nil{
			if err == mongo.ErrNoDocuments{
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return 
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error":  err.Error()})
			return 
		}
		c.JSON(http.StatusOK, user)
	}
 }