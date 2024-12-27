package memory

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

type LLMProcessor struct {
	llm         llms.Model
	mainContext *MemoryContext
	mainProc    chan llms.MessageContent
	functions   []llms.FunctionDefinition
	output      func(llms.MessageContent)
}

func NewLLMProcessor(llm llms.Model, mainContext *MemoryContext) *LLMProcessor {

	saveMsgsfunc := llms.FunctionDefinition{
		Name:        "SaveMessages",
		Description: "SaveMessages",
		Parameters:  map[string]any{},
	}

	return &LLMProcessor{
		llm:         llm,
		mainContext: mainContext,
		mainProc:    make(chan llms.MessageContent),
		functions:   []llms.FunctionDefinition{saveMsgsfunc},
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
				llms.WithFunctions(processor.functions),
			)

			if len(response.Choices[0].ToolCalls) > 0 {
				for _, toolCall := range response.Choices[0].ToolCalls {
					msg.Parts = append(msg.Parts, toolCall)
				}
			}

			msg := llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content)
			processor.mainContext.Messages = append(processor.mainContext.Messages, msg)

			processor.mainProc <- msg
		case llms.ChatMessageTypeAI:
			for _, part := range msg.Parts {
				if toolCall, ok := part.(llms.ToolCall); ok {
					switch toolCall.FunctionCall.Name {
					case "SaveMessages":
						_ = processor.mainContext.SaveMessages()
						msg := llms.TextParts(llms.ChatMessageTypeFunction, "Messages saved")
						processor.mainContext.Messages = append(processor.mainContext.Messages, msg)

						processor.mainProc <- msg
					}
				} else {
					processor.output(msg)
				}
			}
		}
	}
}
