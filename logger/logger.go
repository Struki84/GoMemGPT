package logger

import (
	"fmt"
	"log"

	"github.com/tmc/langchaingo/llms"
)

var Enabled = true

func LogMesages(messages []llms.MessageContent) {
	log.Println("<<<<Short Term Messages>>>>")
	for _, msg := range messages {
		fmt.Println(fmt.Sprintf("%s: %s", msg.Role, msg.Parts[0].(llms.TextContent).String()))
	}
}

func LogLastMessage(messages []llms.MessageContent) {
	lastMessage := messages[len(messages)-1]

	if toolResponse, ok := lastMessage.Parts[0].(llms.ToolCallResponse); ok {
		fmt.Println(fmt.Sprintf("%s: %s", lastMessage.Role, toolResponse.Content))
	} else {
		fmt.Println(fmt.Sprintf("%s: %s", lastMessage.Role, lastMessage.Parts[0].(llms.TextContent).String()))
	}

}
