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
	LoadMessages() ([]llms.MessageContent, error)
	SaveMessages(messages []llms.MessageContent) error

	LoadWorkingContext() (string, error)
	SaveWorkingContext(workingContext string) error

	RecallMessages() ([]llms.MessageContent, error)
	ArchiveMessages(messages []llms.MessageContent) error

	RecallWorkingContext() (string, error)
	ArchiveWorkingContext(workingContext string) error

	SearchMesssgesArchives(query string) ([]llms.MessageContent, error)
	SearchChatHistoryArchives(query string) ([]llms.ChatMessage, error)
}

// Main memory context
type MemoryContext struct {
	// FIFO Message Queue, stores a rolling history of messages,
	// including  messages between the agent and user, as well as system
	// messages (e.g. memory warnings) and function call inputs
	// and outputs. The first index in the FIFO queue stores a system
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
	msgsSize       float32
	workingCtxSize float32
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
		Messages:    make([]llms.MessageContent, 0),
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

// Load short term memory from persistance DB into current memory context
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

func (memory *MemoryContext) Reflect(summary string) error {
	// inputs is working contex. Summary generated
	// by llm based on all the current messsages in context

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

	// we flush all the messages from shrot term memory and leave only the last 3
	// the evicted messages are appended to long term memory
	// this is probably a temp solution
	clanedMsgs := memory.Messages[max(0, len(memory.Messages)-3):]

	err := memory.Storage.ArchiveMessages(memory.Messages)
	if err != nil {
		return err
	}

	err = memory.Storage.SaveMessages(clanedMsgs)
	if err != nil {
		return err
	}

	memory.Messages = clanedMsgs

	err = memory.Storage.SaveWorkingContext(summary)
	if err != nil {
		return err
	}

	memory.WorkingContext = summary

	return nil
}

func (memory *MemoryContext) Recall() error {
	msgs, err := memory.Storage.RecallMessages()
	if err != nil {
		return err
	}

	memory.Messages = msgs

	return nil
}
