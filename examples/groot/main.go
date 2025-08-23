package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/llm/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

const endpoint = "wss://dev-grootafe-pa-googleapis.sandbox.google.com/ws/cloud.ai.groot.afe.GRootAfeService/ExecuteActions"

func main() {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	agent, err := llmagent.New(llmagent.Config{
		Name:        "weather_time_agent",
		Model:       model,
		Description: "Agent to answer questions about the time and weather in a city.",
		Instruction: "I can answer your questions about the time and weather in a city.",
		Tools: []tool.Tool{
			tool.NewGoogleSearchTool(model),
		},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	r, err := runner.NewGRootRunner(&runner.GRootRunnerConfig{
		GRootEndpoint: endpoint,
		GRootAPIKey:   os.Getenv("GROOT_KEY"),
		AppName:       "hello_world",
		RootAgent:     agent,
	})
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nUser -> ")

		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		userMsg := genai.NewContentFromText(userInput, genai.RoleUser)
		fmt.Print("\nAgent -> ")
		for event, err := range r.Run(ctx, "test_user", "test_session", userMsg, &runner.RunConfig{
			StreamingMode: runner.StreamingModeSSE,
		}) {
			if err != nil {
				fmt.Printf("\nAGENT_ERROR: %v\n", err)
			} else {
				for _, p := range event.LLMResponse.Content.Parts {
					fmt.Print(p.Text)
				}
			}
		}
	}
}
