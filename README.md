# GoMemGPT

This library implements the __MemGPT: Towards LLMs as Operating Systems__ paper, which can be found [here](https://arxiv.org/abs/2310.08560).

## The Problem

LLMs have a limited input context, restricting the number of tokens they can process in each inference request. Depending on the model, the context size varies. Since context size influences model performance, most advanced models have a context size between 4K and 16K tokens.

In order to simulate a conversation, a developer needs to manipulate the input context to include conversation history. Due to size limitations, we need a mechanism to manage the input context so that older messages are flushed out while new ones are added. Additionally, flushed messages must be stored somewhere; otherwise, they will be permanently lost from the conversation.

## The Solution

The MemGPT paper proposes two solutions to this problem. First, it suggests treating the LLM as a CPU, the input context as RAM, and database storage as HDD (or SSD). Additionally, the paper proposes storing overflown messages using a combination of a classic SQL database and a vector database.

This solution allows LLMs to have infinite memory, as messages exceeding the input context are stored in databases where they can be retrieved when needed, either via traditional SQL search or similarity search on chat history stored in the vector database.

In this implementation, we create an infinite loop where the system can indefinitely prompt itself, allowing it to "think" about its actions whenever an input signal is detected. The input signal can come from a user or the system itself. Upon receiving input, the system initiates a processing cycle and, depending on the input message, decides what to do next.

The system tracks context size, and when a limit is reached, it triggers a system warning in the processing cycle. This initiates a message flushing and archiving process. During this process, the system creates an abstract of all flushed messages and keeps this abstract in the input context as part of the primary system message, which remains the first message in the message history. Additionally, the system can periodically reflect on the conversation and extract important information to store in the primary system message, ensuring it remains in the input context until no longer needed.

These operations are performed using the LLM's function-calling capabilities. The system is implemented with a set of functions that the LLM can execute to manipulate, edit, and search the input context and stored messages in the database. The LLM decides when to perform these functions.

The current GoMemGPT implementation depends on [LangChainGo](https://github.com/tmc/langchaingo) to handle communication with LLMs and their providers, as well as manage incoming and outgoing messages.

## Usage

This is the first version of the implementation and serves as a proof of concept for the Go implementation of MemGPT. To test it, you need access to an LLM, such as OpenAI's ChatGPT, Anthropic Claude, or a local LLM running on Ollama. You can quickly change the LLM used by modifying the `LLM` variable in the `main.go` file.

### Using Anthropic Claude

**Setup Anthropic API Key**

```bash
export ANTHROPIC_API_KEY=<your_api_key>
```

Set up the LLM to use the Anthropic model:

```go
llm, err := anthropic.New(anthropic.WithModel("claude-3-5-sonnet-latest"))
```

### Using OpenAI

**Setup OpenAI API Key**

```bash
export OPENAI_API_KEY=<your_api_key>
```

Set up the LLM to use an OpenAI model:

```go
llm, err := openai.New(openai.WithModel("gpt-4o"))
```

Once set up, navigate to the root directory and run the following command:

```bash
go run .
```

This will build and run the application, allowing you to start a conversation with the system.

