package main

import "embed"

//go:embed all:routes all:workflows
var configFS embed.FS
