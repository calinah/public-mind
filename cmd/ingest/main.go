package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fmt"
	"public-mind/internal/config"

	"database/sql"
	// Import to register the PostgreSQL driver, even though you don't call its functions directly.
	_ "github.com/lib/pq"

	"github.com/ledongthuc/pdf"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/pgvector/pgvector-go"
)

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
	// Create db client early so we can check if any file has already been processed and stored in db
	db, err := sql.Open("postgres", config.Database.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Check if already processed
	processed, err := isFileProcessed(filePath, db)
	if err != nil {
		return err
	}
	if processed {
		fmt.Printf("  ⏭️  Skipping (already processed): %s\n", filepath.Base(filePath))
		return nil
	}
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
	// Split text into tokens using simple word splitting
	tokens := strings.Fields(text) // Split by whitespace

	var chunks []string

	// Create chunks with overlap
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
}
