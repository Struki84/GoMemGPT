package agent

import (
	"context"
	"log"
	"time"

	"github.com/Struki84/GoMemGPT/memory"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

type ChatAgent struct {
	LLM    llms.Model
	Memory memory.Manager
	Recall llms.Tool
}

func NewChatAgent(llm llms.Model, storage memory.MemoryStorage) *ChatAgent {
	return &ChatAgent{
		LLM: llm,
		Recall: llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "RecallMemory",
				Description: "Recall information from saved long term memory into the current memory context based on current system instruction and user input.",
				Parameters:  map[string]any{},
			},
		},
	}
}

func (agent *ChatAgent) Call(ctx context.Context, memory map[string]any, userMsg llms.MessageContent) (llms.MessageContent, error) {
	memory["time"] = time.Now().Format("January 02, 2006")

	sysPromptTemplate := prompts.PromptTemplate{
		Template:         agent.GetPromptTemplate(),
		TemplateFormat:   prompts.TemplateFormatGoTemplate,
		InputVariables:   []string{},
		PartialVariables: memory,
	}

	sysPrompt, err := sysPromptTemplate.Format(map[string]any{})
	if err != nil {
		log.Printf("Error formatting prompt: %v", err)
		return llms.MessageContent{}, err
	}

	sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, sysPrompt)

	llmResponse, err := agent.LLM.GenerateContent(ctx, []llms.MessageContent{sysMsg, userMsg})
	if err != nil {
		log.Printf("Error generating response: %v", err)
		return llms.MessageContent{}, err
	}

	agentMsg := llms.TextParts(llms.ChatMessageTypeAI, llmResponse.Choices[0].Content)

	if len(llmResponse.Choices[0].ToolCalls) > 0 {
		for _, toolCall := range llmResponse.Choices[0].ToolCalls {
			agentMsg.Parts = append(agentMsg.Parts, toolCall)

			if toolCall.FunctionCall.Name == "RecallContxt" {
				recallResult, err := agent.Memory.RecallMemory(ctx)
				if err != nil {
					log.Printf("Error recalling memory context: %v", err)
					return llms.MessageContent{}, err
				}

				sysPromptTemplate.PartialVariables = agent.Memory.LoadMemory(ctx)

				sysPrompt, err = sysPromptTemplate.Format(map[string]any{})
				if err != nil {
					log.Printf("Error formatting prompt: %v", err)
					return llms.MessageContent{}, err
				}

				sysMsg = llms.TextParts(llms.ChatMessageTypeSystem, sysPrompt)

				llmResponse, err = agent.LLM.GenerateContent(ctx, []llms.MessageContent{sysMsg, userMsg, recallResult})
				if err != nil {
					log.Printf("Error generating recall response: %v", err)
					return llms.MessageContent{}, err
				}
			}
		}
	}

	return agentMsg, nil
}

func (agent *ChatAgent) GetMemory() memory.Manager {
	return agent.Memory
}

func (agent *ChatAgent) GetPromptTemplate() string {
	return `
	{{.time}}
	
	You are a helpful assistant. 

	Your brief history is as follows:
	{{.historicalContext}}

	Your current context is as follows:
	{{.currentContext}}

	Your conversation history with the user is as follows:
	{{.conversationHistory}}
	`
}
