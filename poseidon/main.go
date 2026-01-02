package main

import (
	poseidonfunctions "github.com/jparr721/poseidon-afm/poseidon/agentfunctions"
)

func main() {
	// load up the agent functions directory so all the init() functions execute
	poseidonfunctions.Initialize()
}
