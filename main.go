package main

import (
	"embed"
	"github.com/anonymous-writer/LeaseEditor-Dnsmasq/internal/app"
)

//go:embed templates/index.html static/*
var webAssets embed.FS

func main() { app.Run(webAssets) }
