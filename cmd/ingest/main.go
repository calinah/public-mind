package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"fmt"
	"public-mind/internal/config"

	"database/sql"
	// Import to register the PostgreSQL driver, even though you don't call its functions directly.
	_ "github.com/lib/pq"

	"github.com/gin-gonic/gin"
	"github.com/ledongthuc/pdf"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/pgvector/pgvector-go"
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

func processAllDocuments(config *config.Config) error {
	docsDir := "docs"

	// Get list of PDF files
	files, err := getPDFFiles(docsDir)
	if err != nil {
		return fmt.Errorf("failed to scan docs directory: %w", err)
	}

	fmt.Printf("📁 Found %d PDF files to process\n", len(files))

	// Process each file
	for i, file := range files {
		fmt.Printf("📄 Processing %d/%d: %s\n", i+1, len(files), filepath.Base(file))

		err := processSinglePDF(file, config)
		if err != nil {
			log.Printf("❌ Failed to process %s: %v", file, err)
			continue
		}

		fmt.Printf("✅ Completed: %s\n", filepath.Base(file))
	}

	return nil
}

func getPDFFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func isFileProcessed(fileName string, db *sql.DB) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM documents WHERE file_name = $1", fileName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
func processSinglePDF(filePath string, config *config.Config) error {
	// TODO fix this so there is no processing duplication
	// // Check if already processed
	// processed, err := isFileProcessed(filePath, db)
	// if err != nil {
	// 	return err
	// }
	// if processed {
	// 	fmt.Printf("  ⏭️  Skipping (already processed): %s\n", filepath.Base(filePath))
	// 	return nil
	// }
	// TODO: Implement PDF processing
	fmt.Printf("  🔍 Extracting text from %s\n", filepath.Base(filePath))
	// Open the PDF file
	pdf, reader, err := pdf.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer pdf.Close()

	// Extract text from PDF
	text, err := reader.GetPlainText()
	if err != nil {
		return fmt.Errorf("failed to extract text from PDF: %w", err)
	}
	fmt.Println(text)

	// Read all data from io.Reader
	data, err := io.ReadAll(text)
	if err != nil {
		return err
	}

	// Convert bytes to string
	textString := string(data)
	// TODO: Chunk the text
	chunks := chunkTextWithOverlap(textString, config.App.ChunkSize, config.App.ChunkOverlap)
	fmt.Println(chunks)
	// TODO: Generate embeddings
	embeddings, err := generateEmbeddings(chunks, config)
	if err != nil {
		return err
	}
	fmt.Printf("  ✨ Generated %d embeddings\n", len(embeddings))
	// TODO : Store in db
	// Create db client
	db, err := sql.Open("postgres", config.Database.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Ensure table exists
	err = ensureTableExists(db)
	if err != nil {
		return err
	}

	err = storeInDatabase(chunks, embeddings, filePath, db)
	if err != nil {
		return fmt.Errorf("failed to store in database: %w", err)
	}

	return nil
}

func ensureTableExists(db *sql.DB) error {
	// Enable pgvector extension
	_, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS vector;`)
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Create table if it doesn't exist
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS documents (
            id SERIAL PRIMARY KEY,
            file_name TEXT NOT NULL,
            chunk_text TEXT NOT NULL,
            embedding vector(1536),
            created_at TIMESTAMP DEFAULT NOW()
        );
    `)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create index
	_, err = db.Exec(`
        CREATE INDEX IF NOT EXISTS documents_embedding_idx 
        ON documents USING ivfflat (embedding vector_cosine_ops);
    `)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

func storeInDatabase(chunks []string, embeddings [][]float64, fileName string, db *sql.DB) error {
	for i, chunk := range chunks {
		// Convert []float64 to []float32
		embedding32 := make([]float32, len(embeddings[i]))
		for j, v := range embeddings[i] {
			embedding32[j] = float32(v)
		}

		// Now convert to pgvector format
		embedding := pgvector.NewVector(embedding32)

		_, err := db.Exec(
			"INSERT INTO documents (file_name, chunk_text, embedding) VALUES ($1, $2, $3)",
			fileName,
			chunk,
			embedding,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateEmbeddings(chunks []string, config *config.Config) ([][]float64, error) {
	// generate an embeddings list - since we would have multiple chunks and one embedding for each list we use [][]float64
	var embeddings [][]float64
	// create OpenAi client using API key from config
	client := openai.NewClient(
		option.WithAPIKey(config.OpenAI.APIKey))
	// create a context
	// this is used for API calls (timeout, cancellation, etc.)
	ctx := context.Background()
	for i, chunk := range chunks {
		// Print progress so we know what's happening
		fmt.Printf(" 🔄 Generating embedding %d/%d\n", i+1, len(chunks))

		// Call OpenAI API to create embedding
		// Send the chunk text to OpenAI
		// OpenAI will convert it to an array of numbers
		resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
			Input: openai.EmbeddingNewParamsInputUnion{
				OfArrayOfStrings: []string{chunk},
			}, // The text to convert
			Model: config.OpenAI.EmbeddingModel, // Which model to use (text-embedding-3-small)
		})
		// Check if there was an error
		// If OpenAI API fails, return the error
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding for chunk %d: %w", i, err)
		}
		// Extract the embedding from the response
		// OpenAI returns the embedding in the response
		// We need to get it out and convert it to []float64
		embedding := resp.Data[0].Embedding
		// Add the embedding to our array
		// Now we have one more embedding stored
		embeddings = append(embeddings, embedding)
	}

	// Return all the embeddings
	// We now have an array of embeddings, one for each chunk
	fmt.Printf("  ✅ Generated %d embeddings\n", len(embeddings))
	return embeddings, nil
}

func chunkTextWithOverlap(text string, chunkSize int, overlap int) []string {
	// 1. Split text into tokens using simple word splitting
	tokens := strings.Fields(text) // Split by whitespace

	var chunks []string

	// 2. Create chunks with overlap
	for i := 0; i < len(tokens); i += chunkSize - overlap {
		// Start of chunk
		start := i

		// End of chunk (but don't go past the end)
		end := i + chunkSize
		if end > len(tokens) {
			end = len(tokens)
		}

		// Get tokens for this chunk
		chunkTokens := tokens[start:end]

		// Convert back to text
		chunkText := strings.Join(chunkTokens, " ")
		chunks = append(chunks, chunkText)

		// Move to next chunk (with overlap)
		i += chunkSize - overlap

		// Stop if we've reached the end
		if end >= len(tokens) {
			break
		}
	}

	return chunks
}

func main() {
	// Load configuration
	config, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}
	// Process all documents
	err = processAllDocuments(config)
	if err != nil {
		log.Fatal("Failed to process documents:", err)
	}

	fmt.Println("✅ Ingestion complete!")

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
