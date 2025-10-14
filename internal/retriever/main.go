package main

import (
	"database/sql"
	"fmt"
	"public-mind/internal/config"
	"public-mind/cmd/ingest"
	"github.com/pgvector/pgvector-go"
    "github.com/openai/openai-go/v3"
    "github.com/openai/openai-go/v3/option"
)

// FUNCTION Search(question, topK):
//     // Step 1: Convert question to embedding
//     embedding = generateEmbedding(question)
    
//     // Step 2: Find similar chunks in database
//     similarChunks = findSimilarChunks(embedding, topK)
    
//     // Step 3: Return results
//     RETURN similarChunks
type Retriever struct {
	db *sql.DB 				// db connection ( to search chunks)
	client *openai.Client	// OpenAI client ( to generate question embeddings)
	config *config.Config   // Config (API keys etc)
}

// not using generateEmbeddings func from cmd/ingest because that chunks docs first and it's unnecessary for questions
func (r *Retriever) generateSingleEmbedding(text string) ([]float32, error){
	resp, err := r.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
		Model: r.config.OpenAI.EmbeddingModel
	})
	if err != nil {
		return nil, err
	}

	// convert to float32
	embedding64 := resp.Data[0].Embedding
	embedding32 := make([]float32, len(embedding64))
	for i,v := range embedding64 {
		embedding32[i] = float32(v)
	}
	return embedding32, nil

}
func searchQuestion(question string, topK string ){
	// Convert questions to embedding
	embedding, err := generateSingleEmbedding(question)
	// TODO Find similar chunks in database
    //  similarChunks = findSimilarChunks(embedding, topK)
    
    // TODOD Return results
    // RETURN similarChunks
}

func findSimilarChunks(embedding string, topK string) {
	// Build SQL query for similarity search
	// Execute query with embedding
	// Convert results to Document objects
	// return documents

}


func main(){

}