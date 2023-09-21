package main

import (
	"context"
	"log"

	"github.com/JayJosh846/donationPlatform/controllers"
	"github.com/JayJosh846/donationPlatform/database"
	"github.com/JayJosh846/donationPlatform/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	// "go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	server *gin.Engine
	us     services.UserService
	uc     controllers.UserController
	ctx    context.Context
	userc  *mongo.Collection
)

func init() {
	ctx = context.TODO()

	userc = database.GetUserCollection(database.Client, "Users")
	us = services.Constructor(userc, ctx)
	uc = controllers.Constructor(us)
	server = gin.Default()
}

func main() {
	client := database.ConnectToMongoDB()
	defer database.CloseMongoDBConnection(client)

	server = gin.Default()
	basepath := server.Group("/api/v1")
	uc.UserRoutes(basepath)

	log.Fatal(server.Run(":9090"))
}
