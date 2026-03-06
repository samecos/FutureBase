package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/archplatform/script-service/internal/models"
)

// Engine executes Python scripts
type Engine struct {
	pythonExecutable string
	maxExecutionTime int
	maxMemoryMB      int
	sandboxEnabled   bool
}

// NewEngine creates a new script execution engine
func NewEngine(pythonExec string, maxTime, maxMem int, sandbox bool) *Engine {
	if pythonExec == "" {
		pythonExec = "python3"
	}
	return &Engine{
		pythonExecutable: pythonExec,
		maxExecutionTime: maxTime,
		maxMemoryMB:      maxMem,
		sandboxEnabled:   sandbox,
	}
}

// ExecutionResult contains the result of script execution
type ExecutionResult struct {
	Output        string
	Error         string
	Logs          string
	ExecutionTime int
	MemoryUsage   int64
}

// Execute runs a script with the given input
func (e *Engine) Execute(ctx context.Context, script *models.Script, input map[string]any) (*ExecutionResult, error) {
	startTime := time.Now()

	// Create temporary directory for execution
	tmpDir, err := os.MkdirTemp("", "script-exec-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write script file
	scriptPath := filepath.Join(tmpDir, "script.py")
	if err := os.WriteFile(scriptPath, []byte(script.Code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	// Write input file
	inputPath := filepath.Join(tmpDir, "input.json")
	inputJSON, _ := json.Marshal(input)
	if err := os.WriteFile(inputPath, inputJSON, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input: %w", err)
	}

	// Create wrapper script that handles input/output
	wrapperCode := `
import json
import sys
import traceback

# Read input
with open('input.json', 'r') as f:
    input_data = json.load(f)

# Execute user script
try:
    exec(open('script.py').read(), {'__name__': '__main__', 'INPUT': input_data, 'OUTPUT': {}})
    
    # Try to get OUTPUT if defined
    local_vars = {}
    exec(open('script.py').read(), {'__name__': '__main__', 'INPUT': input_data}, local_vars)
    
    output = local_vars.get('OUTPUT', {})
    if callable(output):
        output = output()
    
    result = {'success': True, 'output': output}
except Exception as e:
    result = {
        'success': False,
        'error': str(e),
        'traceback': traceback.format_exc()
    }

# Write output
with open('output.json', 'w') as f:
    json.dump(result, f)
`
	wrapperPath := filepath.Join(tmpDir, "wrapper.py")
	if err := os.WriteFile(wrapperPath, []byte(wrapperCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write wrapper: %w", err)
	}

	// Execute script
	cmd := exec.CommandContext(ctx, e.pythonExecutable, wrapperPath)
	cmd.Dir = tmpDir

	// Set resource limits
	if e.maxMemoryMB > 0 {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setrlimit: []syscall.Rlimit{
				{Type: syscall.RLIMIT_AS, Cur: uint64(e.maxMemoryMB) * 1024 * 1024, Max: uint64(e.maxMemoryMB) * 1024 * 1024},
			},
		}
	}

	// Capture output
	outputBytes, err := cmd.CombinedOutput()
	executionTime := int(time.Since(startTime).Milliseconds())

	result := &ExecutionResult{
		ExecutionTime: executionTime,
		Logs:          string(outputBytes),
	}

	// Read output
	outputPath := filepath.Join(tmpDir, "output.json")
	outputData, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		if err != nil {
			result.Error = fmt.Sprintf("Execution failed: %v\nOutput: %s", err, string(outputBytes))
		} else {
			result.Error = fmt.Sprintf("Failed to read output: %v", readErr)
		}
		return result, nil
	}

	var outputResult map[string]any
	if err := json.Unmarshal(outputData, &outputResult); err != nil {
		result.Error = fmt.Sprintf("Failed to parse output: %v", err)
		return result, nil
	}

	if success, ok := outputResult["success"].(bool); ok && success {
		outputJSON, _ := json.Marshal(outputResult["output"])
		result.Output = string(outputJSON)
	} else {
		if errMsg, ok := outputResult["error"].(string); ok {
			result.Error = errMsg
		}
		if traceback, ok := outputResult["traceback"].(string); ok {
			result.Logs += "\n" + traceback
		}
	}

	return result, nil
}

// Validate checks if a script is syntactically correct
func (e *Engine) Validate(code string) error {
	cmd := exec.Command(e.pythonExecutable, "-m", "py_compile", "-")
	cmd.Stdin = strings.NewReader(code)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("syntax error: %s", string(output))
	}
	return nil
}

// GetInstalledPackages returns list of installed Python packages
func (e *Engine) GetInstalledPackages() ([]string, error) {
	cmd := exec.Command(e.pythonExecutable, "-m", "pip", "list", "--format=freeze")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var packages []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			packages = append(packages, line)
		}
	}
	return packages, nil
}

// InstallPackage installs a Python package
func (e *Engine) InstallPackage(packageName string) error {
	cmd := exec.Command(e.pythonExecutable, "-m", "pip", "install", packageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install package: %s", string(output))
	}
	return nil
}
