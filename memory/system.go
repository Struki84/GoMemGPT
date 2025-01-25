package memory

import (
	"log"

	"github.com/pkoukk/tiktoken-go"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

var (

	// Templates
	primerAssistantTemplate = `
	{{.time}}
	
	You are a helpful assistant. 

	Your current working context is as follows:
	{{.workingContext}}
	`

	primerMemoryTemplate = `
	{{.time}}
	
	You are an intelligent memory manager. 

	Your brief history is as follows:
	{{.historicalContext}}

	Your current context is as follows:
	{{.currentContext}}

	Your conversation history with the user is as follows:
	{{.messages}
	`

	memoryPressureWorkingContext = `
	{{.time}}
	
	Memory pressure warning: WorkingContext
	
	Working context:
	{{.workingContext}}

	Working context size: {{.workingContextSize}}
	`

	memoryPressureMessages = `
	{{.time}}
	
	Memory pressure warning: Messages
	
	Messages size: {{.messagesSize}}
	`
)

type SystemMonitor struct {
	// The system instructions are readonly (static) and contain information
	// on the MemGPT control flow, the intended usage of the different memory
	// levels, and instructions on how to use the MemGPT functions
	// (e.g. how to retrieve out-of-context data).
	Instructions map[string]string
}

func NewSystemMonitor() *SystemMonitor {
	return &SystemMonitor{
		Instructions: map[string]string{
			"primer:MemoryTemplate":         primerMemoryTemplate,
			"primer:assistantTemplate":      primerAssistantTemplate,
			"memoryPressure:WorkingContext": memoryPressureWorkingContext,
			"memoryPressure:Messages":       memoryPressureMessages,
		},
	}
}

func (system *SystemMonitor) Instruction(instruction string, variables map[string]any) (string, error) {
	template := prompts.PromptTemplate{
		Template:         system.Instructions[instruction],
		TemplateFormat:   prompts.TemplateFormatGoTemplate,
		PartialVariables: variables,
	}

	prompt, err := template.Format(variables)
	if err != nil {
		log.Printf("Error formatting prompt: %v", err)
		return "", err
	}

	return prompt, nil
}

func (system *SystemMonitor) InspectMemoryPressure(agent *Agent) {
	if agent.memory.currentWorkingContextSize() >= int(agent.memory.contextSize*0.9) {
		sysPrompt, err := system.Instruction("memoryPressure:WorkingContext", map[string]any{
			"workingContextSize": agent.memory.currentWorkingContextSize(),
		})

		if err != nil {
			log.Printf("Error formatting prompt: %v", err)
			return
		}

		sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, sysPrompt)
		agent.processor.Input(sysMsg)
	}

	if agent.memory.MessagesTokenSize() >= int(agent.memory.msgsSize*0.9) {
		sysPrompt, err := system.Instruction("memoryPressure:Messages", map[string]any{
			"messagesSize": agent.memory.MessagesTokenSize(),
		})

		if err != nil {
			log.Printf("Error formatting prompt: %v", err)
			return
		}

		sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, sysPrompt)
		agent.processor.Input(sysMsg)
	}
}
