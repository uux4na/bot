package main

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/firestore/apiv1/firestorepb"
	firebase "firebase.google.com/go"
	"github.com/gin-contrib/cors"
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
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
	}))

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ping")
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

		user, userExists := requestBody["profileUrl"]
		reason, reasonExists := requestBody["reason"]

		if !userExists || !reasonExists {
			missingFields := []string{}
			if !userExists {
				missingFields = append(missingFields, "profileUrl")
			}
			if !reasonExists {
				missingFields = append(missingFields, "reason")
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing fields", "fields": missingFields})
			return
		}

		_, _, err := client.Collection("bot-profiles").Add(ctx, map[string]interface{}{
			"url":    user,
			"reason": reason,
		})
		if err != nil {
			log.Printf("An error has occurred: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User successfully added", "profileUrl": user, "reason": reason})
	})
	r.POST("/commentvalid", func(c *gin.Context) {
		var requestBody map[string]string
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		comment, exists := requestBody["comment"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Comment is required"})
			return
		}

		iter := client.Collection("bot-comments").Documents(ctx)
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
			firestoreURL := doc.Data()["comment"].(string)
			if comment == firestoreURL {
				c.JSON(http.StatusOK, gin.H{"valid": true, "message": "Comment found"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"valid": false, "message": "BOT not found"})
	})

	r.POST("/commentadd", func(c *gin.Context) {
		var requestBody map[string]string
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		comment, exists := requestBody["comment"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "comment is required"})
			return
		}

		_, _, err := client.Collection("bot-comments").Add(ctx, map[string]interface{}{
			"comment": comment,
		})
		if err != nil {
			log.Printf("An error has occurred: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"Comment successfully added": comment})
	})

	r.GET("/total", func(c *gin.Context) {
		collection := client.Collection("bot-profiles")
		query := collection.Query

		aggregationQuery := query.NewAggregationQuery().WithCount("all")
		results, err := aggregationQuery.Get(ctx)
		if err != nil {
			log.Printf("Error querying Firestore: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		count, ok := results["all"]
		if !ok {
			log.Printf("Error: couldn't get alias for COUNT from results")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		countValue := count.(*firestorepb.Value)
		c.JSON(http.StatusOK, gin.H{"total": countValue.GetIntegerValue()})
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
