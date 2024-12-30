package memory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

type LLMProcessor struct {
	llm         llms.Model
	mainContext *MemoryContext
	mainProc    chan llms.MessageContent
	functions   []llms.Tool
	output      func(llms.MessageContent)
}

var tools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Load",
			Description: "MemoryContext.Load() will load the last saved state of the memory context into current memory context state.",
			Parameters:  map[string]any{},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Save",
			Description: "MemoryContext.Save() will save the current state of the memory context into presistance db.",
			Parameters:  map[string]any{},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Memorize",
			Description: "MemoryContext.Memorize() will save the current state of the memory context into archive db and clear the overflushed messages and chat history. The input should be the updated summary of the full conversation between human and AI.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary": map[string]any{
						"type":        "string",
						"description": "This should be the updated summary of the full conversation between human and AI.",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Reflect",
			Description: "MemoryContext.Reflect() will save the historical context to archive db and clear the overflushed messages and chat history. The input should be the updated summary of all the internal messages and the conversation between human and AI.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary": map[string]any{
						"type":        "string",
						"description": "This should be the updated summary of the internal messages and conversation between human and AI.",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Compress",
			Description: "MemoryContext.Reflect() will save working context and historical context to core memory, the inputs should be working context and historical context summaries generated from current messages and chat history.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workingContextSummary": map[string]any{
						"type":        "string",
						"description": "This should be the summary of the full conversation between human and AI and the current working context.",
					},

					"historicalContextSummary": map[string]any{
						"type":        "string",
						"description": "This should be the summary of all messages and internal operations and conversations between human and AI and the current historical context.",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Recall",
			Description: "MemoryContext.Recall() uses similarity search to recall information relevany to the current context from archive db.",
			Parameters:  map[string]any{},
		},
	},
}

func NewLLMProcessor(llm llms.Model, mainContext *MemoryContext) *LLMProcessor {

	return &LLMProcessor{
		llm:         llm,
		mainContext: mainContext,
		mainProc:    make(chan llms.MessageContent),
		functions:   tools,
	}
}

func (processor *LLMProcessor) Input(msg llms.MessageContent) {
	processor.mainProc <- msg
}

func (processor *LLMProcessor) Output(fn func(llms.MessageContent)) {
	processor.output = fn
}

func (processor *LLMProcessor) Run() {
	ctx := context.Background()
	for msg := range processor.mainProc {
		switch msg.Role {
		case llms.ChatMessageTypeSystem, llms.ChatMessageTypeFunction:

			response, _ := processor.llm.GenerateContent(
				ctx,
				processor.mainContext.Messages,
				llms.WithTools(processor.functions),
			)

			newMsg := llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content)

			if len(response.Choices[0].ToolCalls) > 0 {
				for _, toolCall := range response.Choices[0].ToolCalls {
					newMsg.Parts = append(msg.Parts, toolCall)
				}
			}

			processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
			processor.mainProc <- msg
		case llms.ChatMessageTypeAI:
			for _, part := range msg.Parts {
				if toolCall, ok := part.(llms.ToolCall); ok {
					switch toolCall.FunctionCall.Name {
					case "Load":
						err := processor.mainContext.Load()
						if err != nil {
							newMsg := llms.TextParts(llms.ChatMessageTypeFunction, fmt.Sprintf("Error loading context: %v", err))

							processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
							processor.mainProc <- msg
						}

						msg := llms.TextParts(llms.ChatMessageTypeFunction, "Messages loaded")

						processor.mainContext.Messages = append(processor.mainContext.Messages, msg)
						processor.mainProc <- msg
					case "Save":
						err := processor.mainContext.Save()
						if err != nil {
							newMsg := llms.TextParts(llms.ChatMessageTypeFunction, fmt.Sprintf("Error saving context: %v", err))

							processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
							processor.mainProc <- msg
						}
						newMsg := llms.TextParts(llms.ChatMessageTypeFunction, "Messages saved")

						processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
						processor.mainProc <- msg
					case "Compress":
						var input struct {
							WorkingContextSummary    string `json:"workingContextSummary"`
							HistoricalContextSummary string `json:"historicalContextSummary"`
						}

						err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &input)
						if err != nil {
							newMsg := llms.TextParts(llms.ChatMessageTypeFunction, fmt.Sprintf("Error parsing parameters: %v", err))

							processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
							processor.mainProc <- msg
						}

						err = processor.mainContext.Compress(input.WorkingContextSummary, input.HistoricalContextSummary)
						if err != nil {
							newMsg := llms.TextParts(llms.ChatMessageTypeFunction, fmt.Sprintf("Error compressing context: %v", err))

							processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
							processor.mainProc <- msg
						}

						newMsg := llms.TextParts(llms.ChatMessageTypeFunction, "Context compressed")

						processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
						processor.mainProc <- msg
					case "Memorize":
						var input struct {
							Summary string `json:"summary"`
						}

						err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &input)
						if err != nil {
							newMsg := llms.TextParts(llms.ChatMessageTypeFunction, fmt.Sprintf("Error parsing parameters: %v", err))

							processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
							processor.mainProc <- msg
						}

						err = processor.mainContext.Memorize(input.Summary)
						if err != nil {
							newMsg := llms.TextParts(llms.ChatMessageTypeFunction, fmt.Sprintf("Error memorizing context: %v", err))

							processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
							processor.mainProc <- msg
						}

						newMsg := llms.TextParts(llms.ChatMessageTypeFunction, "Context memorized")

						processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
						processor.mainProc <- msg

					case "Reflect":
						var input struct {
							Summary string `json:"summary"`
						}

						err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &input)
						if err != nil {
							newMsg := llms.TextParts(llms.ChatMessageTypeFunction, fmt.Sprintf("Error parsing parameters: %v", err))

							processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
							processor.mainProc <- msg
						}

						err = processor.mainContext.Reflect(input.Summary)
						if err != nil {
							newMsg := llms.TextParts(llms.ChatMessageTypeFunction, fmt.Sprintf("Error reflecting on context: %v", err))

							processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
							processor.mainProc <- msg
						}

						newMsg := llms.TextParts(llms.ChatMessageTypeFunction, "Context reflected")

						processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
						processor.mainProc <- msg
					}
				} else {
					processor.output(msg)
				}
			}
		}
	}
}
