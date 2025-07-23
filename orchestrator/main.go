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
			model_id TEXT,
			output TEXT,
			latency FLOAT,
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
			Prompt  string `json:"prompt"`
			Task    string `json:"task"`
			ModelID string `json:"model_id"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		inferPayload := map[string]string{
			"prompt":   req.Prompt,
			"task":     req.Task,
			"model_id": req.ModelID,
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
			Output    string  `json:"output"`
			Latency   float64 `json:"latency"`
			ModelUsed string  `json:"model_used"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&inferResp); err != nil {
			log.Printf("Failed to decode inference response: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid inference response"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, err = db.Exec(ctx,
			`INSERT INTO jobs (prompt, task, model_id, output, latency) VALUES ($1, $2, $3, $4, $5)`,
			req.Prompt, req.Task, inferResp.ModelUsed, inferResp.Output, inferResp.Latency,
		)
		if err != nil {
			log.Printf("Failed to insert job: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store job"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Job stored and inference completed",
			"prompt":   req.Prompt,
			"task":     req.Task,
			"model_id": inferResp.ModelUsed,
			"output":   inferResp.Output,
			"latency":  inferResp.Latency,
		})
	})

	router.GET("/jobs", func(c *gin.Context) {
		rows, err := db.Query(context.Background(), `
			SELECT id, prompt, task, model_id, output, latency, created_at
			FROM jobs
			ORDER BY id DESC
			LIMIT 20
		`)
		if err != nil {
			log.Printf("Failed to fetch jobs: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs"})
			return
		}
		defer rows.Close()

		type Job struct {
			ID        int       `json:"id"`
			Prompt    string    `json:"prompt"`
			Task      string    `json:"task"`
			ModelID   string    `json:"model_id"`
			Output    string    `json:"output"`
			Latency   float64   `json:"latency"`
			CreatedAt time.Time `json:"created_at"`
		}

		var jobs []Job

		for rows.Next() {
			var job Job
			err := rows.Scan(&job.ID, &job.Prompt, &job.Task, &job.ModelID, &job.Output, &job.Latency, &job.CreatedAt)
			if err != nil {
				log.Printf("Failed to scan row: %v\n", err)
				continue
			}
			jobs = append(jobs, job)
		}

		c.JSON(http.StatusOK, jobs)
	})

	router.Run(":8000")
}
