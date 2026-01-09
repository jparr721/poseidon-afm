package main

import (
	"github.com/MythicMeta/MythicContainer"

	// Import agentfunctions to trigger init() registration of commands and payload definition
	_ "github.com/jparr721/poseidon-afm/poseidon/agentfunctions"
)

func main() {
	MythicContainer.StartAndRunForever(nil)
}
