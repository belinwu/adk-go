package main

import (
	"log"

	"google.golang.org/adk/internal/graph"
	"google.golang.org/adk/llm"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

func filter(input []*session.Event) ([]*session.Event, error) {
	event := session.NewEvent("invocation.todo")
	event.LLMResponse = &llm.Response{
		Content: genai.Text("Filtered prompt")[0],
	}
	return []*session.Event{event}, nil
}

func search(input []*session.Event) ([]*session.Event, error) {
	event := session.NewEvent("invocation.todo")
	event.LLMResponse = &llm.Response{
		Content: genai.Text("Here are the search results from Google:")[0],
	}
	return []*session.Event{event}, nil
}

func codeSearch(input []*session.Event) ([]*session.Event, error) {
	event := session.NewEvent("invocation.todo")
	event.LLMResponse = &llm.Response{
		Content: genai.Text("Here is the generated code...")[0],
	}
	return []*session.Event{event}, nil
}

func generate(input []*session.Event) ([]*session.Event, error) {
	event := session.NewEvent("invocation.todo")
	event.LLMResponse = &llm.Response{
		Content: genai.Text("Let me explain you in summary what I know...")[0],
	}
	return []*session.Event{event}, nil
}

func main() {
	ex := graph.NewExecutor()
	if err := ex.Start(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	ex.Spawn(&graph.Node{
		Inputs: []string{"prompt"},
		Output: "filtered_prompt",
		Run:    filter,
	})
	ex.Spawn(&graph.Node{
		Inputs: []string{"filtered_prompt"},
		Output: "search_results",
		Run:    search,
	})
	ex.Spawn(&graph.Node{
		Inputs: []string{"filtered_prompt"},
		Output: "code_results",
		Run:    codeSearch,
	})
	ex.Spawn(&graph.Node{
		Inputs: []string{"code_results", "search_results", "filtered_prompt"},
		Output: "final_response",
		Run:    generate,
	})

	ex.Put("prompt", []*session.Event{{
		LLMResponse: &llm.Response{
			Content: genai.Text("Can you describe me what I can do with this code?")[0],
		},
	}})

	ex.Wait() // blocks until everything is executed.

	log.Println("\n\nReplaying from event log...")
	replayer := graph.NewReplayExecutor(ex.EventLog())
	replayer.Start()
}
