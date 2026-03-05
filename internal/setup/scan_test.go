package setup

import (
	"path/filepath"
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixture(name string) string {
	return filepath.Join("testdata", "fixtures", name)
}

// --- Framework detection ---

func TestDetectFramework_LangGraphRequirements(t *testing.T) {
	fw := detectFramework(fixture("langgraph_project"))
	assert.Equal(t, agentfile.FrameworkLangGraph, fw)
}

func TestDetectFramework_LangGraphPyproject(t *testing.T) {
	fw := detectFramework(fixture("pyproject_project"))
	assert.Equal(t, agentfile.FrameworkLangGraph, fw)
}

func TestDetectFramework_Generic(t *testing.T) {
	fw := detectFramework(fixture("generic_project"))
	assert.Equal(t, agentfile.FrameworkGeneric, fw)
}

func TestDetectFramework_FastAPIIsGeneric(t *testing.T) {
	fw := detectFramework(fixture("fastapi_project"))
	assert.Equal(t, agentfile.FrameworkGeneric, fw)
}

func TestDetectFramework_EmptyProject(t *testing.T) {
	fw := detectFramework(fixture("empty_project"))
	assert.Equal(t, agentfile.FrameworkGeneric, fw)
}

func TestDetectFramework_NonexistentDir(t *testing.T) {
	fw := detectFramework("/nonexistent/dir")
	assert.Equal(t, agentfile.FrameworkGeneric, fw)
}

func TestDetectFramework_LangGraphPipfile(t *testing.T) {
	fw := detectFramework(fixture("pipfile_langgraph"))
	assert.Equal(t, agentfile.FrameworkLangGraph, fw)
}

func TestDetectFramework_PipenvGeneric(t *testing.T) {
	fw := detectFramework(fixture("pipenv_project"))
	assert.Equal(t, agentfile.FrameworkGeneric, fw)
}

// --- Entry point detection ---

func TestDetectEntryPoint_MainPy(t *testing.T) {
	entry, warn := detectEntryPoint(fixture("fastapi_project"))
	assert.Equal(t, "main.py", entry)
	assert.Nil(t, warn)
}

func TestDetectEntryPoint_PriorityOrder(t *testing.T) {
	entry, warn := detectEntryPoint(fixture("multi_entry"))
	assert.Equal(t, "main.py", entry)
	assert.Nil(t, warn)
}

func TestDetectEntryPoint_NoMatch(t *testing.T) {
	entry, warn := detectEntryPoint(fixture("empty_project"))
	assert.Equal(t, "main.py", entry)
	require.NotNil(t, warn)
	assert.Contains(t, warn.What, "No entry point detected")
	assert.Equal(t, "main.py", warn.Assumed)
}

func TestDetectEntryPoint_NonexistentDir(t *testing.T) {
	entry, warn := detectEntryPoint("/nonexistent/dir")
	assert.Equal(t, "main.py", entry)
	require.NotNil(t, warn)
}

// --- Port detection ---

func TestDetectPort_UvicornRun(t *testing.T) {
	code := `uvicorn.run(app, host="0.0.0.0", port=8080)`
	port, warn := detectPort(code)
	assert.Equal(t, 8080, port)
	assert.Nil(t, warn)
}

func TestDetectPort_DotRun(t *testing.T) {
	code := `app.run(port=5000)`
	port, warn := detectPort(code)
	assert.Equal(t, 5000, port)
	assert.Nil(t, warn)
}

func TestDetectPort_CLIFlag(t *testing.T) {
	code := `# uvicorn main:app --port 3000`
	port, warn := detectPort(code)
	assert.Equal(t, 3000, port)
	assert.Nil(t, warn)
}

func TestDetectPort_Default(t *testing.T) {
	code := `print("hello")`
	port, warn := detectPort(code)
	assert.Equal(t, 8000, port)
	require.NotNil(t, warn)
	assert.Contains(t, warn.What, "Port not detected")
	assert.Equal(t, "8000", warn.Assumed)
}

func TestDetectPort_EmptyCode(t *testing.T) {
	port, warn := detectPort("")
	assert.Equal(t, 8000, port)
	require.NotNil(t, warn)
}

// --- Health path detection ---

func TestDetectHealth_AppGet(t *testing.T) {
	code := `@app.get("/health")\ndef health(): return {"ok": True}`
	path, warn := detectHealthPath(code)
	assert.Equal(t, "/health", path)
	assert.Nil(t, warn)
}

func TestDetectHealth_AppRoute(t *testing.T) {
	code := `@app.route("/healthcheck")\ndef health(): pass`
	path, warn := detectHealthPath(code)
	assert.Equal(t, "/healthcheck", path)
	assert.Nil(t, warn)
}

func TestDetectHealth_RouterGet(t *testing.T) {
	code := `@router.get("/healthz")\nasync def healthz(): return {"ok": True}`
	path, warn := detectHealthPath(code)
	assert.Equal(t, "/healthz", path)
	assert.Nil(t, warn)
}

func TestDetectHealth_Default(t *testing.T) {
	code := `print("hello")`
	path, warn := detectHealthPath(code)
	assert.Equal(t, "/health", path)
	require.NotNil(t, warn)
	assert.Contains(t, warn.What, "Health endpoint not detected")
}

func TestDetectHealth_EmptyCode(t *testing.T) {
	path, warn := detectHealthPath("")
	assert.Equal(t, "/health", path)
	require.NotNil(t, warn)
}

// --- Env var detection ---

func TestDetectEnvVars_OsEnvironBracket(t *testing.T) {
	vars := detectEnvVars(fixture("fastapi_project"))
	assert.Contains(t, vars, "OPENAI_API_KEY")
	assert.Contains(t, vars, "MODEL_NAME")
}

func TestDetectEnvVars_OsEnvironGet(t *testing.T) {
	vars := detectEnvVars(fixture("langgraph_project"))
	assert.Contains(t, vars, "ANTHROPIC_API_KEY")
	assert.Contains(t, vars, "DATABASE_URL")
}

func TestDetectEnvVars_OsGetenv(t *testing.T) {
	vars := detectEnvVars(fixture("generic_project"))
	assert.Contains(t, vars, "DB_HOST")
}

func TestDetectEnvVars_EmptyProject(t *testing.T) {
	vars := detectEnvVars(fixture("empty_project"))
	assert.Nil(t, vars)
}

func TestDetectEnvVars_Sorted(t *testing.T) {
	vars := detectEnvVars(fixture("fastapi_project"))
	require.Len(t, vars, 2)
	assert.True(t, vars[0] < vars[1], "env vars should be sorted: got %v", vars)
}

func TestDetectEnvVars_NoDuplicates(t *testing.T) {
	vars := detectEnvVars(fixture("fastapi_project"))
	seen := make(map[string]bool)
	for _, v := range vars {
		assert.False(t, seen[v], "duplicate env var: %s", v)
		seen[v] = true
	}
}

// --- ScanProject integration ---

func TestScanProject_FastAPI(t *testing.T) {
	result := ScanProject(fixture("fastapi_project"))
	assert.Equal(t, agentfile.FrameworkGeneric, result.Framework)
	assert.Equal(t, "main.py", result.EntryPoint)
	assert.Equal(t, 8080, result.Port)
	assert.Equal(t, "/health", result.HealthPath)
	assert.Contains(t, result.EnvVars, "OPENAI_API_KEY")
	assert.Empty(t, result.Warnings)
}

func TestScanProject_LangGraph(t *testing.T) {
	result := ScanProject(fixture("langgraph_project"))
	assert.Equal(t, agentfile.FrameworkLangGraph, result.Framework)
	assert.Contains(t, result.EnvVars, "ANTHROPIC_API_KEY")
	// No main.py in langgraph_project — expect entry point warning + port + health warnings
	assert.True(t, len(result.Warnings) >= 1)
}

func TestScanProject_Generic(t *testing.T) {
	result := ScanProject(fixture("generic_project"))
	assert.Equal(t, agentfile.FrameworkGeneric, result.Framework)
	assert.Equal(t, "main.py", result.EntryPoint)
	assert.Equal(t, 8000, result.Port)   // default
	assert.Equal(t, "/health", result.HealthPath) // default
	assert.Contains(t, result.EnvVars, "DB_HOST")
}

func TestScanProject_EmptyProject(t *testing.T) {
	result := ScanProject(fixture("empty_project"))
	assert.Equal(t, agentfile.FrameworkGeneric, result.Framework)
	assert.Equal(t, "main.py", result.EntryPoint) // default
	assert.Equal(t, 8000, result.Port)
	assert.Equal(t, "/health", result.HealthPath)
	assert.Nil(t, result.EnvVars)
	assert.True(t, len(result.Warnings) >= 1) // at least entry point warning
}

func TestScanProject_Pyproject(t *testing.T) {
	result := ScanProject(fixture("pyproject_project"))
	assert.Equal(t, agentfile.FrameworkLangGraph, result.Framework)
	assert.Equal(t, "main.py", result.EntryPoint)
	assert.Equal(t, 9000, result.Port)
	assert.Equal(t, "/healthz", result.HealthPath)
	assert.Contains(t, result.EnvVars, "SECRET_KEY")
	assert.Empty(t, result.Warnings)
}

func TestScanProject_PoetryProject(t *testing.T) {
	result := ScanProject(fixture("poetry_project"))
	assert.Equal(t, agentfile.FrameworkLangGraph, result.Framework)
	assert.Equal(t, agentfile.PackageManagerPoetry, result.PackageManager)
	assert.Equal(t, "main.py", result.EntryPoint)
}

func TestScanProject_UVProject(t *testing.T) {
	result := ScanProject(fixture("uv_project"))
	assert.Equal(t, agentfile.FrameworkGeneric, result.Framework)
	assert.Equal(t, agentfile.PackageManagerUV, result.PackageManager)
}

func TestScanProject_PipenvProject(t *testing.T) {
	result := ScanProject(fixture("pipenv_project"))
	assert.Equal(t, agentfile.FrameworkGeneric, result.Framework)
	assert.Equal(t, agentfile.PackageManagerPipenv, result.PackageManager)
}

func TestScanProject_PipDefault(t *testing.T) {
	result := ScanProject(fixture("fastapi_project"))
	assert.Equal(t, agentfile.PackageManagerPip, result.PackageManager)
}

// --- Package name extraction ---

func TestExtractPackageName(t *testing.T) {
	cases := []struct {
		line     string
		expected string
	}{
		{"langgraph>=0.1.0", "langgraph"},
		{"fastapi==0.100.0", "fastapi"},
		{"requests", "requests"},
		{"uvicorn[standard]>=0.23.0", "uvicorn"},
		{"pydantic~=2.0", "pydantic"},
		{"openai!=1.0.0", "openai"},
		{"  flask  ", "flask"},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			assert.Equal(t, tc.expected, extractPackageName(tc.line))
		})
	}
}
