package sica

import "embed"

//go:embed web/static/*
var StaticFiles embed.FS
