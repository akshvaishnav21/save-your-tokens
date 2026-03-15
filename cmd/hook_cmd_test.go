package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessHookInput_HappyPath(t *testing.T) {
	input := `{"tool_name":"Bash","tool_input":{"command":"git log -10","description":"Show commits"}}`
	result := processHookInput([]byte(input), nil)
	require.NotNil(t, result)

	var out hookOutput
	require.NoError(t, json.Unmarshal(result, &out))
	assert.Equal(t, "allow", out.PermissionDecision)
	assert.Equal(t, "syt git log -10", out.UpdatedInput.Command)
	assert.Equal(t, "Show commits", out.UpdatedInput.Description)
}

func TestProcessHookInput_NonBashTool(t *testing.T) {
	input := `{"tool_name":"Read","tool_input":{"command":"git log -10"}}`
	assert.Nil(t, processHookInput([]byte(input), nil))
}

func TestProcessHookInput_AlreadySyt(t *testing.T) {
	input := `{"tool_name":"Bash","tool_input":{"command":"syt git log -10"}}`
	assert.Nil(t, processHookInput([]byte(input), nil))
}

func TestProcessHookInput_Multiline(t *testing.T) {
	input := "{\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"git log\\ngit status\"}}"
	assert.Nil(t, processHookInput([]byte(input), nil))
}

func TestProcessHookInput_EmptyCommand(t *testing.T) {
	input := `{"tool_name":"Bash","tool_input":{"command":""}}`
	assert.Nil(t, processHookInput([]byte(input), nil))
}

func TestProcessHookInput_NoRegistryMatch(t *testing.T) {
	input := `{"tool_name":"Bash","tool_input":{"command":"echo hello"}}`
	assert.Nil(t, processHookInput([]byte(input), nil))
}

func TestProcessHookInput_InvalidJSON(t *testing.T) {
	assert.Nil(t, processHookInput([]byte("not json"), nil))
}

func TestProcessHookInput_EmptyInput(t *testing.T) {
	assert.Nil(t, processHookInput([]byte(""), nil))
}

func TestProcessHookInput_ExcludeCommands(t *testing.T) {
	input := `{"tool_name":"Bash","tool_input":{"command":"git log -10"}}`
	result := processHookInput([]byte(input), []string{"git log"})
	assert.Nil(t, result)
}
