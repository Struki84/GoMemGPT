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

	memoryPressureWorkingContext = `
	{{.time}}
	
	Memory pressure warning: WorkingContext

	Rewrite your working context so it takes up less space.

	Working context size: {{.workingContextSize}}
	`

	memoryPressureMessages = `
	{{.time}}
	
	Memory pressure warning: Messages

	Move messages from your short term memory context to your long term memory context and save a sumary of your messages to your short term memory working context.
	
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
		"time":           time.Now().Format("January 02, 2006, 15:04:05"),
		"workingContext": system.mainContext.WorkingContext,
	})

	if err != nil {
		log.Printf("Error formatting prompt: %v", err)
		return err
	}

	// log.Printf("System primer prompt: %s", primerPrompt)

	primerMsg := llms.TextParts(llms.ChatMessageTypeSystem, primerPrompt)

	if len(system.mainContext.Messages) == 0 {
		system.mainContext.Messages = []llms.MessageContent{primerMsg}
	} else {
		if system.mainContext.Messages[0].Role == llms.ChatMessageTypeSystem {
			system.mainContext.Messages[0] = primerMsg
		} else {
			system.mainContext.Messages = append([]llms.MessageContent{primerMsg}, system.mainContext.Messages...)
		}
	}

	system.mainContext.Messages = append(system.mainContext.Messages, msg)

	return nil
}
