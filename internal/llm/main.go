package main

import (
	"fmt"
	"log"

	// "errors"
	"context"
	"public-mind/internal/config"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}
	fmt.Println(config.OpenAI.APIKey)
	client := openai.NewClient(
		option.WithAPIKey(config.OpenAI.APIKey))
	ctx := context.Background()
	question := "how do openAI go clients work?"

	resp, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(question)},
		Model: config.OpenAI.CompletionModel,
	})
	if err != nil {
		log.Fatal("Error calling OpenAI:", err)
	}
	fmt.Println(resp.OutputText())

}
