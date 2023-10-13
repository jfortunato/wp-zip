package packager

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"syscall"
)

type Prompter interface {
	Prompt(question string) string
}

// RuntimePrompter is a Prompter that prompts the user at runtime for input.
type RuntimePrompter struct{}

// Prompt prompts the user with the given question and returns their response.
func (p *RuntimePrompter) Prompt(question string) string {
	fmt.Println(question)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatalln(err)
	}

	return response
}

// PromptForPassword prompts the user for input without echoing the input to the terminal.
func (p *RuntimePrompter) PromptForPassword(question string) string {
	fmt.Println(question)
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalln(err)
	}

	return string(password)
}
