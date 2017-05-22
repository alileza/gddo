package main

import "os"

func init() {
	// Register an oauth token at https://github.com/settings/tokens/new
	oAuthToken = os.Getenv("GITHUB_TOKEN")
}
