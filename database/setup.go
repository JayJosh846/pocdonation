package database

import (
	"context"
	"fmt"
	"log"
	"os"

	// "go.mongodb.org/mongo-driver/bson"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectToMongoDB() *mongo.Client {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	uri := os.Getenv("DATABASE_URL")

	// Define MongoDB connection options.
	clientOptions := options.Client().ApplyURI(uri)

	// Create a MongoDB client.
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	// Check if the connection was successful.
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil
	}

	fmt.Println("Connected to MongoDB!")

	return client
}

var Client *mongo.Client = ConnectToMongoDB()

func GetUserCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	// Get a reference to the database and collection.
	database := client.Database("Pocdonation")
	collection := database.Collection(collectionName)

	return collection
}

func CloseMongoDBConnection(client *mongo.Client) {
	if client != nil {
		client.Disconnect(context.Background())
		fmt.Println("Disconnected from MongoDB.")
	}
}
