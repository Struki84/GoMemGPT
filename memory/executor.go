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
			Name:        "Recall",
			Description: "MemoryContext.Recall() uses similarity search to recall information relevany to the current context from archive db.",
			Parameters:  map[string]any{},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "InternalOutput",
			Description: "",
			Parameters:  map[string]any{},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "ExternalOutput",
			Description: "",
			Parameters:  map[string]any{},
		},
	},
}

// Execute processor instructions based on the llm function selection
// and preforms operations on the memory context
type Executor struct {
	operator  MemoryOperator
	functions []llms.Tool
}

func NewExecutor(mainContext *MemoryContext) Executor {
	return Executor{
		operator:  *NewMemoryOperator(mainContext),
		functions: functions,
	}
}

// Run the llm functions
func (executor *Executor) Run(fn llms.ToolCall) (string, error) {
	switch fn.FunctionCall.Name {
	case "Load":
		err := executor.operator.Load()
		if err != nil {
			return "", err
		}

		return "Memory context loaded", nil
	case "Save":
		err := executor.operator.Save()
		if err != nil {
			return "", err
		}

		return "Memory context saved", nil
	case "Memorize":
		err := executor.operator.Memorize(fn.FunctionCall.Arguments)
		if err != nil {
			return "", err
		}

		return "Memory context memorized", nil
	case "Reflect":
		err := executor.operator.Reflect(fn.FunctionCall.Arguments)
		if err != nil {
			return "", err
		}

		return "Memory context reflected", nil
	case "Recall":
		err := executor.operator.Recall()
		if err != nil {
			return "", err
		}

		return "Memory context recalled", nil
	case "InternalOutput":
		return executor.operator.InternalOutput(fn.FunctionCall.Arguments), nil
	case "ExternalOutput":
		return executor.operator.ExternalOutput(fn.FunctionCall.Arguments), nil
	}

	return "", nil
}
