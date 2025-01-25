package memory

import (
	"fmt"
	"log"

	"github.com/pkoukk/tiktoken-go"
	"github.com/tmc/langchaingo/llms"
)

var (
	defaultContextSize    float32 = 4096
	defualtMsgsSize       float32 = 0.7
	defaultWorkingCtxSize float32 = 0.3
)

type MemoryStorage interface {
	LoadShortTermMemory() ([]llms.MessageContent, error)
	SaveShortTermMemory(messages []llms.MessageContent) error

	LoadLongTermMemory() ([]llms.MessageContent, error)
	SaveLongTermMemory(messages []llms.MessageContent) error

	LoadMessages() ([]llms.MessageContent, error)
	SaveMessages(messages []llms.MessageContent) error

	LoadChatHistory() ([]llms.ChatMessage, error)
	SaveChatHistory(chatHistory []llms.ChatMessage) error

	LoadWorkingContext() (string, error)
	SaveWorkingContext(workingContext string) error

	LoadHistoricalContext() (string, error)
	SaveHistoricalContext(historicalContext string) error

	RecallMessages() ([]llms.MessageContent, error)
	ArchiveMessages(messages []llms.MessageContent) error

	RecallChatHistory() ([]llms.ChatMessage, error)
	ArchiveChatHistory(chatHistory []llms.ChatMessage) error

	RecallWorkingContext() (string, error)
	ArchiveWorkingContext(workingContext string) error

	RecallHistoricalContext() (string, error)
	ArchiveHistoricalContext(historicalContext string) error

	SearchMesssgesArchives(query string) ([]llms.MessageContent, error)
	SearchChatHistoryArchives(query string) ([]llms.ChatMessage, error)
}

// Main context

// 1. Needs to be able to load short term memeory into curent context from a persistance DB
// 2. Needs to be able to save current context into persistance DB
type MemoryContext struct {
	// FIFO Message Queue, stores a rolling history of messages,
	// including  messages between the agent and user, as well as system
	// messages (e.g. memory warnings) and function call inputs
	// and outputs. The first indehx in the FIFO queue stores a system
	// message containing a recursive summary of messages that have
	// been evicted from the queue.
	Messages []llms.MessageContent

	// Current working context
	// Working context is a fixed-size read/write block of unstructured text,
	// writeable only via MemGPT function calls.
	WorkingContext string

	// Intarface for perfomring operations on the data storage
	Storage MemoryStorage

	contextSize    float32
	workingCtxSize float32
	msgsSize       float32
}

// MemoryContext can be viewd as state, or core memory with
// perisistance through a db of any kind,

// The MemoryContext needs to be able to load previous conversations
// internal messages, and context into it's state by rebuilding it
// more persisttance storage and archival storage.

// When chat history queue is full, the oldest messages are evicted
// from the queue and summerized into current working context.

// core memory - fixed size memory context with its state saved in persistance storage db
// archive memory - unlimited size db storage for all messages and contexts

func NewMemoryContext(storage MemoryStorage) *MemoryContext {
	return &MemoryContext{
		Storage:     storage,
		contextSize: defaultContextSize,
	}
}

func (memory *MemoryContext) CurrentWorkingContextSize() int {
	encoder, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		log.Printf("Error creating tiktoken encoder: %v", err)
		return 0
	}

	return len(encoder.Encode(memory.WorkingContext, nil, nil))
}

func (memory *MemoryContext) CurrentMessagesSize() int {
	totalTokens := 0
	encoder, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		log.Printf("Error creating tiktoken encoder: %v", err)
		return 0
	}

	combineAllTextParts := func(parts []llms.ContentPart) string {
		result := ""
		for _, part := range parts {
			switch v := part.(type) {
			case llms.TextContent:
				result += v.String()
			case llms.ToolCall:
				txt := fmt.Sprintf("%s %v", v.FunctionCall.Name, v.FunctionCall.Arguments)
				result += txt
			default:
				// ignore or handle other types
			}
		}
		return result
	}

	for _, msg := range memory.Messages {
		contentToEncode := fmt.Sprintf("%s: %s", msg.Role, combineAllTextParts(msg.Parts))
		tokenIDs := encoder.Encode(contentToEncode, nil, nil)
		totalTokens += len(tokenIDs)
	}

	// The chat format typically has an extra 2 tokens at the end
	totalTokens += 2

	return totalTokens
}

// Load all the messages and context from core memory
func (memory *MemoryContext) Load() error {
	msgs, err := memory.Storage.LoadMessages()
	if err != nil {
		return err
	}

	workingContext, err := memory.Storage.LoadWorkingContext()
	if err != nil {
		return err
	}

	memory.Messages = msgs
	memory.WorkingContext = workingContext

	return nil
}

// Save current memory context state to core memory
func (memory *MemoryContext) Save() error {

	err := memory.Storage.SaveMessages(memory.Messages)
	if err != nil {
		return err
	}

	err = memory.Storage.SaveWorkingContext(memory.WorkingContext)
	if err != nil {
		return err
	}

	return nil
}

// Summarize or compress memories into working context or historical context
// to save space
func (memory *MemoryContext) Compress(summary string) error {
	// inputs are working and historical Summary generated
	// by llm, based on chat history and messsages
	// updates memory.WorkingContext and memory.HistoricalContext
	// saves the memory context state to core memory

	memory.WorkingContext = summary

	err := memory.Storage.SaveWorkingContext(memory.WorkingContext)
	if err != nil {
		return err
	}

	return nil
}

// Move infromation from core memory to archive memory
func (memory *MemoryContext) Memorize(summary string) error {
	// can happen when chat history is full
	// save chat history msgs to archive storage
	// removes overflowing messages in chat history
	// saves the new summary to core memory and archive memory
	// input should be the summary of the flushed chat history messages

	err := memory.FlushMessages()
	if err != nil {
		return err
	}

	err = memory.Storage.ArchiveWorkingContext(memory.WorkingContext)
	if err != nil {
		return err
	}

	memory.WorkingContext = summary

	err = memory.Storage.SaveWorkingContext(memory.WorkingContext)
	if err != nil {
		return err
	}

	return nil
}

// Generate internal thoughts about the context
// func (memory *MemoryContext) Reflect(args string) error {
// can happen when messages is full
// save messages to archive storage
// removes overflowing messages in messages
// saves the new summary to core memory and archive memory
// input should be the summary of the flushed messages

// 	var input struct {
// 		Summary string `json:"summary"`
// 	}
//
// 	err := json.Unmarshal([]byte(args), &input)
// 	if err != nil {
// 		return err
// 	}
//
// 	err = memory.FlushMessages()
// 	if err != nil {
// 		return err
// 	}
//
// 	memory.HistoricalContext = input.Summary
//
// 	err = memory.Storage.SaveHistoricalContext(memory.HistoricalContext)
// 	if err != nil {
// 		return err
// 	}
//
// 	err = memory.Storage.ArchiveHistoricalContext(memory.HistoricalContext)
// 	if err != nil {
// 		return err
// 	}
//
// 	return nil
// }

func (memory *MemoryContext) Archive() error {
	err := memory.Storage.ArchiveWorkingContext(memory.WorkingContext)
	if err != nil {
		return err
	}

	err = memory.Storage.ArchiveMessages(memory.Messages)
	if err != nil {
		return err
	}

	return nil
}

// Recall information from archive storage
func (memory *MemoryContext) Recall() error {
	workingContext, err := memory.Storage.RecallWorkingContext()
	if err != nil {
		return err
	}

	msgs, err := memory.Storage.RecallMessages()
	if err != nil {
		return err
	}

	memory.Messages = msgs
	memory.WorkingContext = workingContext

	return nil
}

func (memory *MemoryContext) Search() error {
	// use similarity search to recall information from archive storage
	// input will be tbd

	return nil
}

func (memory *MemoryContext) FlushMessages() error {
	// while the token size of all messages in buffer
	// is equal or greater than the max token size  minus the current input msg(?)
	// keep removing messages from messages buffer

	return nil

}

func (memory *MemoryContext) FlushChatHistory() error {
	// while the token size of all chat history messages in buffer
	// is equal or greater than the max token size  minus the last input msg(?)
	// keep removing messages from chat history buffer

	return nil
}
