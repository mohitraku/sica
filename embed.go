package sica

import "embed"

//go:embed web/static/*
var StaticFiles embed.FS

//go:embed web/templates/*
var TemplateFiles embed.FS
