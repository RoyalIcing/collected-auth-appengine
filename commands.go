package main

import (
	"context"
	"fmt"
	"strings"
)

// A Command can be run to see a result
type Command interface {
	Run(ctx context.Context) (CommandResult, error)
}

// ParseCommandInput parses a /… command
func ParseCommandInput(input string) (Command, error) {
	input = strings.TrimLeft(input, "/")
	request := strings.SplitN(input, "\n", 2)

	if len(request) == 0 {
		return nil, fmt.Errorf("No command passed")
	}

	var params string
	if len(request) >= 2 {
		params = request[1]
	}

	commands := parseSubcommands(request[0])

	return parseCommand(commands, params)
}

func parseSubcommands(input string) []string {
	inputCommands := strings.Split(input, " ")
	output := make([]string, 0, len(inputCommands))
	for _, c := range inputCommands {
		c = strings.TrimSpace(c)
		output = append(output, c)
	}
	return output
}

func parseCommand(commands []string, params string) (Command, error) {
	if len(commands) == 0 {
		return nil, fmt.Errorf("No command passed")
	}

	if commands[0] == "web" && len(commands) >= 2 {
		return ParseWebCommand(commands[1:], params)
	}

	return nil, fmt.Errorf("Unknown command %v", commands)
}
