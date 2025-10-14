package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"public-mind/internal/config"

	"github.com/gin-gonic/gin"
)

type AskRequest struct {
	Question string `json:"question"`
}

func handleAsk(c *gin.Context) {
	var req AskRequest

	// parse and validate JSON body
	if err := c.ShouldBindJSON(&req); err != nil {
		// return bad request?
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}
	if req.Question == "" {
		// handle error and send back a bad return(400)
		c.JSON(http.StatusBadRequest, gin.H{"error": "question is required"})
		// stop processing the request
		return
	}
	// Process the question
	answer := "This is where you'd call the LLM to process the question"

	// Respond with JSON
	c.JSON(http.StatusOK, gin.H{"answer": answer})
}

func main() {
	// Load configuration
	config, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Create a Gin router with default middleware (logger and recovery)
	r := gin.Default()

	// Define a simple GET endpoint
	r.GET("/healthz", func(c *gin.Context) {
		// Return JSON response
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	r.POST("/ask", handleAsk)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.Server.Port),
		Handler: r,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
