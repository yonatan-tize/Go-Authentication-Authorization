package database

import(
	"log"
	"time"
	"os"
	"context"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DBinstance() *mongo.Client{
	 err := godotenv.Load(".env")
	 if err != nil{
		log.Fatal("Error loading the .env file")
	 }
	 MongoDb := os.Getenv("MONGODB_URL")

	 clientOption := options.Client().ApplyURI(MongoDb)
	 //set timeout 
	 ctx, cancel:= context.WithTimeout(context.Background(), 10*time.Second)
	 defer cancel()

	 client, err := mongo.Connect(ctx, clientOption)
	 if err != nil{
		log.Fatal("failed to connect to database", err)
	 }

	 err = client.Ping(ctx, nil)
	 if err != nil{
		log.Fatal("failed to connect to database")
	 }

	 return client
}

var Client *mongo.Client = DBinstance()

func OpenCollection(Client *mongo.Client, collectionName string) *mongo.Collection{
	var collection *mongo.Collection = Client.Database("go-auth").Collection(collectionName)
	return collection
}
