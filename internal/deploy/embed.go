package deploy

import "embed"

//go:embed templates/*.tmpl
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS
