package main

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	ctx context.Context
	app *firebase.App
)

const (
	firebaseConfigFile = "./firebase.json"
)

func setupRouter(client *firestore.Client) *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.POST("/urlcheck", func(c *gin.Context) {
		var requestBody map[string]string
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		url, exists := requestBody["url"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
			return
		}

		iter := client.Collection("bot-profiles").Documents(ctx)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Error querying Firestore: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
				return
			}
			firestoreURL := doc.Data()["url"].(string)
			if url == firestoreURL {
				c.JSON(http.StatusOK, gin.H{"valid": true, "message": "BOT found"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"valid": false, "message": "BOT not found"})
	})

	r.POST("/addprofile", func(c *gin.Context) {
		var requestBody map[string]string
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, exists := requestBody["profileUrl"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "profileUrl is required"})
			return
		}

		_, _, err := client.Collection("bot-profiles").Add(ctx, map[string]interface{}{
			"url": user,
		})
		if err != nil {
			log.Printf("An error has occurred: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"User successfully added": user})
	})

	return r
}

func main() {
	ctx = context.Background()
	sa := option.WithCredentialsFile("./firebase.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Error initializing Firestore client: %v", err)
	}
	defer client.Close()

	r := setupRouter(client)
	r.Run(":8080")
}
