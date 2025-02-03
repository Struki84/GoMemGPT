package memory

import (
	"github.com/tmc/langchaingo/llms"
)

var functions = []llms.Tool{
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

type Executor struct {
	mainContext *MemoryContext
	functions   []llms.Tool
}

func NewExecutor(mainContext *MemoryContext) Executor {

	return Executor{
		mainContext: mainContext,
		functions:   functions,
	}
}

func (executor *Executor) Run(fn llms.ToolCall) (string, error) {
	switch fn.FunctionCall.Name {
	case "Load":
		err := executor.mainContext.Load()
		if err != nil {
			return "", err
		}

		return "Memory context loaded", nil
	case "Save":
		err := executor.mainContext.Save()
		if err != nil {
			return "", err
		}

		return "Memory context saved", nil
	// case "Compress":
	// 	err := executor.mainContext.Compress(fn.FunctionCall.Arguments)
	// 	if err != nil {
	// 		return "", err
	// 	}
	//
	// 	return "Memory context compressed", nil

	case "Memorize":
		err := executor.mainContext.Memorize(fn.FunctionCall.Arguments)
		if err != nil {
			return "", err
		}

		return "Memory context memorized", nil
	case "Reflect":
		err := executor.mainContext.Reflect(fn.FunctionCall.Arguments)
		if err != nil {
			return "", err
		}

		return "Memory context reflected", nil
	case "Recall":
		err := executor.mainContext.Recall()
		if err != nil {
			return "", err
		}

		return "Memory context recalled", nil
	}

	return "", nil
}
