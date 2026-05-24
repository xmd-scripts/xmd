package config

import (
	"os"
	"strings"
)

// Config holds runtime configuration from environment variables and CLI flags.
type Config struct {
	// Backend selection
	Backend string // openai_completion, agent_command, cursor, gemini

	// openai_completion backend settings
	CompletionURL    string
	CompletionModel  string
	CompletionAPIKey string

	// agent_command backend settings
	AgentCmd string

	// Sandbox settings
	NoSandbox   bool
	AllowRead   []string
	AllowWrite  []string

	// Context file for conversation persistence (JSONL)
	ContextFile string

	// Debug prints the rendered prompt to stderr before running
	Debug bool
}

// FromEnv loads configuration from environment variables.
func FromEnv() *Config {
	c := &Config{}

	c.Backend = getEnv("XMD_BACKEND", "openai_completion")
	c.CompletionURL = getEnv("XMD_COMPLETION_URL", "http://localhost:11434/v1/chat/completions")
	c.CompletionModel = os.Getenv("XMD_COMPLETION_MODEL")
	c.CompletionAPIKey = os.Getenv("XMD_COMPLETION_API_KEY")
	c.AgentCmd = os.Getenv("XMD_AGENT_CMD")

	c.Debug = os.Getenv("XMD_DEBUG") != ""

	if v := os.Getenv("XMD_ALLOW_READ"); v != "" {
		c.AllowRead = append(c.AllowRead, strings.Split(v, ":")...)
	}
	if v := os.Getenv("XMD_ALLOW_WRITE"); v != "" {
		c.AllowWrite = append(c.AllowWrite, strings.Split(v, ":")...)
	}

	return c
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

