package main

import (
	"context"
	"database/sql"
	"fmt"
	"public-mind/internal/config"

	_ "github.com/lib/pq"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/pgvector/pgvector-go"
)

type Retriever struct {
	db     *sql.DB        // db connection ( to search chunks)
	client openai.Client  // OpenAI client ( to generate question embeddings)
	config *config.Config // Config (API keys etc)
}

type Document struct {
	ID         int
	FileName   string
	ChunkText  string
	Similarity float64
}

// not using generateEmbeddings func from cmd/ingest because that chunks docs first and it's unnecessary for questions
func (r *Retriever) generateSingleEmbedding(text string) ([]float32, error) {
	ctx := context.Background()
	resp, err := r.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
		Model: r.config.OpenAI.EmbeddingModel,
	})
	if err != nil {
		return nil, err
	}

	// convert to float32
	embedding64 := resp.Data[0].Embedding
	embedding32 := make([]float32, len(embedding64))
	for i, v := range embedding64 {
		embedding32[i] = float32(v)
	}
	return embedding32, nil

}
func (r *Retriever) searchQuestion(question string, topK int) ([]Document, error) {
	// Convert questions to embedding
	embedding, err := r.generateSingleEmbedding(question)
	if err != nil {
		return nil, err
	}

	// Find similar chunks in database
	similarChunks, err := r.findSimilarChunks(embedding, topK)
	if err != nil {
		return nil, err
	}

	// Return results
	return similarChunks, nil
}

func (r *Retriever) findSimilarChunks(embedding []float32, topK int) ([]Document, error) {
	// Build SQL query for vector similarity search
	// <=> is the cosine distance operator in pgvector
	// 1 - distance converts distance to similarity score (higher = more similar)
	// ORDER BY embedding <=> $1 sorts by similarity (most similar first)
	// LIMIT $2 restricts results to topK most similar chunks
	query := `SELECT id, file_name, chunk_text, 1 - (embedding <=> $1) as similarity FROM documents ORDER BY embedding <=> $1 LIMIT $2`

	// Execute the query with the question embedding and topK limit
	// pgvector.NewVector() wraps the embedding for PostgreSQL
	rows, err := r.db.Query(query, pgvector.NewVector(embedding), topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // Always close the rows to free database resources

	// Create slice to store the retrieved documents
	var docs []Document

	// Iterate through each row returned by the query
	for rows.Next() {
		var doc Document

		// Scan the row data into the Document struct
		// Fields must match the SELECT order: id, file_name, chunk_text, similarity
		err := rows.Scan(&doc.ID, &doc.FileName, &doc.ChunkText, &doc.Similarity)
		if err != nil {
			return nil, err
		}

		// Add the document to our results slice
		docs = append(docs, doc)
	}

	// Return the list of similar documents
	return docs, nil
}

func main() {
	// Load configuration
	config, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	// Connect to database
	db, err := sql.Open("postgres", config.Database.URL)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}
	defer db.Close()

	// Create OpenAI client
	client := openai.NewClient(
		option.WithAPIKey(config.OpenAI.APIKey))

	// Create retriever instance
	retriever := &Retriever{
		db:     db,
		client: client,
		config: config,
	}

	// Test the retriever with a sample question
	question := "What are the development permit areas?"
	topK := 3

	fmt.Printf("🔍 Searching for: %s\n", question)

	results, err := retriever.searchQuestion(question, topK)
	if err != nil {
		fmt.Printf("❌ Search failed: %v\n", err)
		return
	}

	// Display results
	fmt.Printf("✅ Found %d similar chunks:\n\n", len(results))
	for i, doc := range results {
		fmt.Printf("--- Result %d ---\n", i+1)
		fmt.Printf("File: %s\n", doc.FileName)
		fmt.Printf("Similarity: %.3f\n", doc.Similarity)
		fmt.Printf("Text: %s\n\n", doc.ChunkText[:min(200, len(doc.ChunkText))])
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
