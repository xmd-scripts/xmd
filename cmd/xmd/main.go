package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xmd-scripts/xmd/internal/backend"
	"github.com/xmd-scripts/xmd/internal/config"
	"github.com/xmd-scripts/xmd/internal/prompt"
	"github.com/xmd-scripts/xmd/internal/sandbox"
	"github.com/xmd-scripts/xmd/internal/script"
	"github.com/xmd-scripts/xmd/internal/tool"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cfg := config.FromEnv()

	var (
		scriptPath  string
		kvArgs      []string
		showHelp    bool
		showVersion bool
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--version" || arg == "-v":
			showVersion = true
		case arg == "--help" || arg == "-h":
			showHelp = true
		case arg == "--debug":
			cfg.Debug = true
		case arg == "--no-sandbox":
			cfg.NoSandbox = true
		case arg == "--context" && i+1 < len(args):
			i++
			cfg.ContextFile = args[i]
		case strings.HasPrefix(arg, "--context="):
			cfg.ContextFile = strings.TrimPrefix(arg, "--context=")
		case arg == "--backend" && i+1 < len(args):
			i++
			cfg.Backend = args[i]
		case strings.HasPrefix(arg, "--backend="):
			cfg.Backend = strings.TrimPrefix(arg, "--backend=")
		case arg == "--allow-read" && i+1 < len(args):
			i++
			cfg.AllowRead = append(cfg.AllowRead, args[i])
		case arg == "--allow-write" && i+1 < len(args):
			i++
			cfg.AllowWrite = append(cfg.AllowWrite, args[i])
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(os.Stderr, "xmd: unknown flag: %s\n", arg)
			return 2
		case scriptPath == "" && !strings.Contains(arg, "="):
			scriptPath = arg
		case strings.Contains(arg, "="):
			kvArgs = append(kvArgs, arg)
		default:
			fmt.Fprintf(os.Stderr, "xmd: unexpected argument: %s\n", arg)
			return 2
		}
	}

	if showVersion {
		fmt.Println(version)
		return 0
	}

	if showHelp && scriptPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: xmd SCRIPT [key=value ...] [--flags]\n")
		fmt.Fprintf(os.Stderr, "       ./SCRIPT [key=value ...] [--flags]\n")
		return 0
	}

	if cfg.Debug {
		if err := os.Setenv("XMD_DEBUG", "1"); err != nil {
			fmt.Fprintf(os.Stderr, "xmd: failed to set debug env: %v\n", err)
			return 1
		}
	}

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "xmd: cannot determine working directory: %v\n", err)
		return 1
	}

	if cfg.ContextFile != "" {
		absCtx, err := filepath.Abs(cfg.ContextFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "xmd: cannot resolve context file path: %v\n", err)
			return 1
		}
		// Resolve symlinks on the parent dir so the sandbox policy matches the
		// real path (on macOS /tmp -> /private/tmp).
		if real, err := filepath.EvalSymlinks(filepath.Dir(absCtx)); err == nil {
			absCtx = filepath.Join(real, filepath.Base(absCtx))
		}
		cfg.AllowWrite = append(cfg.AllowWrite, absCtx)
		cfg.ContextFile = absCtx
	}

	if !cfg.NoSandbox && os.Getenv("XMD_SANDBOXED") != "1" {
		return runInSandbox(cfg, pwd, os.Args, func() int {
			return runScript(cfg, pwd, scriptPath, kvArgs, showHelp)
		})
	}

	return runScript(cfg, pwd, scriptPath, kvArgs, showHelp)
}

func runScript(cfg *config.Config, pwd, scriptPath string, kvArgs []string, showHelp bool) int {
	scriptData, absPath, isFragment, err := loadScript(scriptPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	var s *script.Script
	if isFragment {
		s, err = script.ParseFragment(absPath, scriptData)
	} else {
		s, err = script.Parse(absPath, scriptData)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	supplied, err := script.ParseCLIVars(kvArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	resolved, err := script.Resolve(s)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	if showHelp {
		printHelp(s, resolved)
		return 0
	}

	for name, decl := range resolved.Vars {
		if !decl.Stdin {
			continue
		}
		stat, err := os.Stdin.Stat()
		if err != nil || stat.Mode()&os.ModeCharDevice != 0 {
			fmt.Fprintf(os.Stderr, "xmd: vars: variable '%s' requires stdin input (pipe or redirect)\n", name)
			return 2
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "xmd: vars: failed to read stdin: %s\n", err)
			return 2
		}
		supplied[name] = strings.TrimRight(string(data), "\n")
		break
	}

	boundVars, err := script.BindVars(resolved.Vars, supplied)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	fullPrompt := prompt.BuildVarsBlock(boundVars, resolved.Body)

	if cfg.Debug {
		fmt.Fprintln(os.Stderr, "=== rendered prompt ===")
		fmt.Fprintln(os.Stderr, fullPrompt)
		fmt.Fprintln(os.Stderr, "=== end ===")
	}

	if runErr := runBackend(cfg, cfg.ContextFile, tool.Message{Role: s.Role, Content: fullPrompt}); runErr != nil {
		fmt.Fprintln(os.Stderr, runErr.Error())
		return 1
	}
	return 0
}

func loadScript(path string) (data []byte, absPath string, fragment bool, err error) {
	if path == "" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			err = fmt.Errorf("xmd: failed to read stdin: %w", err)
		}
		return data, "<stdin>", true, err
	}
	data, err = os.ReadFile(path)
	if err != nil {
		return nil, "", false, fmt.Errorf("xmd: script: file not found: %s", path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, "", false, fmt.Errorf("xmd: cannot resolve script path: %w", err)
	}
	return data, abs, false, nil
}

func runBackend(cfg *config.Config, contextID string, prompt tool.Message) error {
	b, err := newBackend(cfg)
	if err != nil {
		return err
	}
	return b.Run(contextID, prompt, os.Stdout, os.Stderr)
}

func newBackend(cfg *config.Config) (backend.Backend, error) {
	switch cfg.Backend {
	case "openai_completion":
		b := backend.NewOpenAICompletion(cfg.CompletionURL, cfg.CompletionModel, cfg.CompletionAPIKey)
		if cfg.Debug {
			b.DebugOut = os.Stderr
		}
		return b, nil
	case "agent_command":
		if cfg.AgentCmd == "" {
			return nil, fmt.Errorf("xmd: backend: XMD_AGENT_CMD not set")
		}
		return backend.NewAgentCommand(cfg.AgentCmd), nil
	default:
		return nil, fmt.Errorf("xmd: backend: unknown backend '%s'", cfg.Backend)
	}
}

func runInSandbox(cfg *config.Config, pwd string, argv []string, fallback func() int) int {
	tempDir, err := os.MkdirTemp("", "xmd-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "xmd: sandbox: policy setup failed: %s\n", err)
		return 2
	}
	defer os.RemoveAll(tempDir)

	var (
		sb    sandbox.Sandbox
		sbErr error
	)
	if cfg.NoSandbox {
		sb = sandbox.NewNoSandbox()
		fmt.Fprintln(os.Stderr, "xmd: WARNING: sandboxing is disabled")
	} else {
		sb, sbErr = sandbox.New()
		if sbErr != nil {
			fmt.Fprintln(os.Stderr, sbErr.Error())
			return 2
		}
	}

	allowRead := cfg.AllowRead
	if exe, err := os.Executable(); err == nil {
		allowRead = append(allowRead, filepath.Dir(exe))
	}

	wrappedArgv, err := sb.Wrap(argv, pwd, tempDir, allowRead, cfg.AllowWrite)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	if len(wrappedArgv) == 0 {
		return fallback()
	}

	var stderrBuf bytes.Buffer
	cmd := exec.Command(wrappedArgv[0], wrappedArgv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Env = append(os.Environ(), "XMD_SANDBOXED=1")

	if err := cmd.Run(); err != nil {
		// On macOS, sandbox-exec requires the parent sandbox to permit sandbox_apply.
		// If it doesn't, fall back to running without a nested policy — the parent's
		// constraints still apply.
		if strings.Contains(stderrBuf.String(), "sandbox_apply") {
			fmt.Fprintln(os.Stderr, "xmd: warning: sandbox policy could not be applied (sandbox_apply denied by parent); running under parent sandbox constraints")
			return fallback()
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

func printHelp(s *script.Script, resolved *script.ResolvedScript) {
	if s.Description != "" {
		fmt.Printf("Description: %s\n", s.Description)
	} else {
		fmt.Printf("Script: %s\n", s.Path)
	}
	fmt.Printf("Role: %s\n", s.Role)
	if len(resolved.Vars) == 0 {
		fmt.Println("Variables: none")
		return
	}
	fmt.Println("\nVariables:")
	for name, decl := range resolved.Vars {
		switch {
		case decl.Stdin:
			fmt.Printf("  %-20s (stdin)\n", name)
		case decl.Required:
			fmt.Printf("  %-20s (required)\n", name)
		default:
			fmt.Printf("  %-20s (default: %q)\n", name, decl.Default)
		}
	}
}
