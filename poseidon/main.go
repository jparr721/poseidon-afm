package main

import (
	poseidonfunctions "MyContainer/poseidon/agentfunctions"
)

func main() {
	// load up the agent functions directory so all the init() functions execute
	poseidonfunctions.Initialize()
}
