package repos_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateScaffoldRepo_Scaffold(t *testing.T) {
	tempDir := t.TempDir()
	repo := repos.NewTemplateScaffoldRepo(tempDir)

	err := repo.Scaffold()
	require.NoError(t, err)

	// Verify AGENTS.md was created
	agentsFile := filepath.Join(tempDir, "AGENTS.md")
	assert.FileExists(t, agentsFile)

	content, err := os.ReadFile(agentsFile)
	require.NoError(t, err)

	contentStr := string(content)

	// Verify no absolute file URLs
	assert.NotContains(t, contentStr, "file://")

	// Verify relative skill paths are used
	assert.Contains(t, contentStr, "[bibx-inquire SKILL.md](.agents/skills/bibx-inquire/SKILL.md)")
	assert.Contains(t, contentStr, "[bibx-query SKILL.md](.agents/skills/bibx-query/SKILL.md)")
	assert.Contains(t, contentStr, "[bibx-search SKILL.md](.agents/skills/bibx-search/SKILL.md)")

	// Verify primary agent description is generic
	assert.Contains(t, contentStr, "The primary agent initiates the implementation plans")
}
