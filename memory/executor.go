package memory

import (
	"encoding/json"

	"github.com/tmc/langchaingo/llms"
)

var functions = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Load",
			Description: "Load will load your short term memory context from presistance db.",
			Parameters:  map[string]any{},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Save",
			Description: "Save will save your short term memory context into presistance db.",
			Parameters:  map[string]any{},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Memorize",
			Description: "Memorize will save the current messages into your long term memory, clear the messages from your short term memory leaving the last 3 messages, and updated the short term memory working context with the summary of evicted messages.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary": map[string]any{
						"type":          "string",
						"descriptionhj": "Summary of all the messages in your current short term memory.",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Reflect",
			Description: "Reflect will save and summarize vital information from the messages in your short term memory or recovered from long term memory and save it into your short term memory working context.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary": map[string]any{
						"type":        "string",
						"description": "Summary of vital information found in current short term messages or messages retreived from ling term memory.",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Recall",
			Description: "Recall will fetch a history of your previous conversations with the user and loaded in to a single messages added to your short term context, if the recalled messages overflow your short term memory context you will receive a warrning.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"search": map[string]any{
						"type":        "string",
						"description": "Query to search for previous conversations.",
					},
					"limit": map[string]any{
						"type":        "number",
						"description": "Number of messages to recall per page.",
					},
					"page": map[string]any{
						"type":        "number",
						"description": "Page number to recall.",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "Think",
			Description: "Think allows you to messages your self and thus enables you to reason about your actions and decisions, you can call think multiple times.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"thought": map[string]any{
						"type":        "string",
						"description": "Internal message to your self.",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "InternalOutput",
			Description: "InternalOutput will end function execution cycle and store the final message into your short term memory context without displaying to the user.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"finalOutput": map[string]any{
						"type":        "string",
						"description": "Message to store into your short term memory context without displaying to the user.",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "ExternalOutput",
			Description: "ExternalOutput will end function execution cycle and store the final message into your short term memory context and display that message to the user.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"finalOutput": map[string]any{
						"type":        "string",
						"description": "Message to store into your short term memory context and display to the user.",
					},
				},
			},
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
		var args struct {
			Summary string `json:"summary"`
		}

		if err := json.Unmarshal([]byte(fn.FunctionCall.Arguments), &args); err != nil {
			return "Error unmarshalling Memorize arguments", err
		}

		err := executor.operator.Memorize(args.Summary)
		if err != nil {
			return "", err
		}

		return "Memory context memorized", nil
	case "Reflect":
		var args struct {
			Summary string `json:"summary"`
		}

		if err := json.Unmarshal([]byte(fn.FunctionCall.Arguments), &args); err != nil {
			return "Error unmarshalling Reflect arguments", err
		}

		err := executor.operator.Reflect(args.Summary)
		if err != nil {
			return "", err
		}

		return "Memory context reflected", nil
	case "Recall":
		var args struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
			Page  int    `json:"page"`
		}

		if err := json.Unmarshal([]byte(fn.FunctionCall.Arguments), &args); err != nil {
			return "Error unmarshalling Recall arguments", err
		}

		err := executor.operator.Recall(args.Query, args.Limit, args.Page)
		if err != nil {
			return "", err
		}

		return "Conversation history recalled", nil
	case "Think":
		var args struct {
			Thought string `json:"thought"`
		}

		if err := json.Unmarshal([]byte(fn.FunctionCall.Arguments), &args); err != nil {
			return "Error unmarshalling Think arguments", err
		}

		return executor.operator.Think(args.Thought), nil
	case "InternalOutput":
		var args struct {
			FinalOutput string `json:"finalOutput"`
		}

		if err := json.Unmarshal([]byte(fn.FunctionCall.Arguments), &args); err != nil {
			return "Error unmarshalling InternalOutput arguments", err
		}

		return executor.operator.InternalOutput(args.FinalOutput), nil
	case "ExternalOutput":
		var args struct {
			FinalOutput string `json:"finalOutput"`
		}

		if err := json.Unmarshal([]byte(fn.FunctionCall.Arguments), &args); err != nil {
			return "Error unmarshalling ExternalOutput arguments", err
		}

		return executor.operator.ExternalOutput(args.FinalOutput), nil
	}

	return "", nil
}
