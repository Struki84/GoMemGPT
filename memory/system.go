package memory

import (
	"log"
	"time"

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

	Working context size: {{.workingContextSize}}
	`

	memoryPressureMessages = `
	{{.time}}
	
	Memory pressure warning: Messages
	
	Messages size: {{.messagesSize}}
	`
)

type SystemMonitor struct {
	mainContext *MemoryContext
	// The system instructions are readonly (static) and contain information
	// on the MemGPT control flow, the intended usage of the different memory
	// levels, and instructions on how to use the MemGPT functions
	// (e.g. how to retrieve out-of-context data).
	Instructions map[string]string
}

func NewSystemMonitor(mainContext *MemoryContext) *SystemMonitor {
	return &SystemMonitor{
		mainContext: mainContext,
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

func (system *SystemMonitor) InspectMemoryPressure() []llms.MessageContent {
	warrnings := []llms.MessageContent{}

	if system.mainContext.CurrentWorkingContextSize() >= int(system.mainContext.workingCtxSize*0.9) {
		sysPrompt, err := system.Instruction("memoryPressure:WorkingContext", map[string]any{
			"workingContextSize": system.mainContext.CurrentWorkingContextSize(),
		})

		if err != nil {
			log.Printf("Error formatting prompt: %v", err)
			return warrnings
		}

		sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, sysPrompt)

		warrnings = append(warrnings, sysMsg)
	}

	if system.mainContext.CurrentMessagesSize() >= int(system.mainContext.msgsSize*0.9) {
		sysPrompt, err := system.Instruction("memoryPressure:Messages", map[string]any{
			"messagesSize": system.mainContext.CurrentMessagesSize(),
		})

		if err != nil {
			log.Printf("Error formatting prompt: %v", err)
			return warrnings
		}

		sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, sysPrompt)

		warrnings = append(warrnings, sysMsg)
	}

	return warrnings
}

func (system *SystemMonitor) AppendMessage(msg llms.MessageContent) error {
	primerPrompt, err := system.Instruction("primer:assistantTemplate", map[string]any{
		"time":           time.Now().Format("January 02, 2006"),
		"workingContext": system.mainContext.WorkingContext,
	})

	if err != nil {
		log.Printf("Error formatting prompt: %v", err)
		return err
	}

	// log.Printf("System primer prompt: %s", primerPrompt)

	primerMsg := llms.TextParts(llms.ChatMessageTypeSystem, primerPrompt)

	system.mainContext.Messages[0] = primerMsg
	system.mainContext.Messages = append(system.mainContext.Messages, msg)

	return nil
}
