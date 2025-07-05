package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func connectToDB() *pgx.Conn {
	dsn := "postgres://modelbench:password@postgres:5432/modelbench"
	var conn *pgx.Conn
	var err error

	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err = pgx.Connect(ctx, dsn)
		if err == nil {
			log.Println("Connected to Postgres!")
			break
		}

		log.Printf("Postgres not ready, retrying in 2s... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	_, err = conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS jobs (
			id SERIAL PRIMARY KEY,
			prompt TEXT NOT NULL,
			task TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create jobs table: %v\n", err)
	}
	log.Println("Table 'jobs' is ready.")

	return conn
}

func main() {
	db := connectToDB()
	defer db.Close(context.Background())

	router := gin.Default()

	router.POST("/submit", func(c *gin.Context) {
		var req struct {
			Prompt string `json:"prompt"`
			Task   string `json:"task"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		// Call the inference service with retries
		inferPayload := map[string]string{
			"prompt": req.Prompt,
			"task":   req.Task,
		}
		jsonData, _ := json.Marshal(inferPayload)

		var resp *http.Response
		var err error
		for i := 0; i < 5; i++ {
			resp, err = http.Post("http://inference:9000/infer", "application/json", bytes.NewBuffer(jsonData))
			if err == nil {
				break
			}
			log.Printf("Retrying inference request (%d/5): %v", i+1, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			log.Printf("Failed to call inference service: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Inference service error"})
			return
		}
		defer resp.Body.Close()

		var inferResp struct {
			Output string `json:"output"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&inferResp); err != nil {
			log.Printf("Failed to decode inference response: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid inference response"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, err = db.Exec(ctx,
			`INSERT INTO jobs (prompt, task) VALUES ($1, $2)`,
			req.Prompt, req.Task,
		)
		if err != nil {
			log.Printf("Failed to insert job: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store job"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Job stored and inference completed",
			"prompt":  req.Prompt,
			"task":    req.Task,
			"output":  inferResp.Output,
		})
	})

	// Run the API server
	router.Run(":8000")
}
