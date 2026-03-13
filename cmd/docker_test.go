package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_DockerPs(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/docker_ps_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterDockerPs(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 20.0,
		"docker ps savings %.1f%% < required 20%%", savings)
}

func TestFilterDockerPs_Empty(t *testing.T) {
	out := filterDockerPs("", "")
	assert.NotEmpty(t, out)
}

func TestFilterDockerPs_NoContainers(t *testing.T) {
	input := "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES\n"
	out := filterDockerPs(input, "")
	assert.NotEmpty(t, out)
}

func TestFilterDockerPs_WithContainers(t *testing.T) {
	input := `CONTAINER ID   IMAGE              COMMAND                  CREATED         STATUS         PORTS                    NAMES
a1b2c3d4e5f6   postgres:15        "docker-entrypoint.s…"   2 hours ago     Up 2 hours     0.0.0.0:5432->5432/tcp   myapp_postgres
b2c3d4e5f6a7   redis:7-alpine     "docker-entrypoint.s…"   2 hours ago     Up 2 hours     0.0.0.0:6379->6379/tcp   myapp_redis
`
	out := filterDockerPs(input, "")
	assert.Contains(t, out, "postgres")
	assert.Contains(t, out, "redis")
}
