package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"PostmanJanai/internal/entity"

	"github.com/stretchr/testify/require"
)

type stubPMJExec struct{}

func (stubPMJExec) Execute(ctx context.Context, in *entity.HTTPExecuteInput) (*entity.HTTPExecuteResult, error) {
	return &entity.HTTPExecuteResult{
		StatusCode:   204,
		ResponseBody: "{}",
	}, nil
}

type noopPMJEnv struct{}

func (noopPMJEnv) ActiveVariableMap(ctx context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (noopPMJEnv) UpsertActiveVariable(ctx context.Context, key, value string) (bool, error) {
	return true, nil
}

func (noopPMJEnv) DeleteActiveVariable(ctx context.Context, key string) (bool, error) {
	return true, nil
}

func TestRunPMJScript_emptyNoExecutorNeeded(t *testing.T) {
	art, err := RunPMJScript(context.Background(), true, "\n ", time.Second,
		nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, art)
	require.Empty(t, art.Console)
	require.Empty(t, art.Tests)
}

func TestRunPMJScript_variablesSession(t *testing.T) {
	session := map[string]string{}
	_, err := RunPMJScript(context.Background(), true, `pmj.variables.set('k', 'v');`, time.Second,
		nil, nil, session, noopPMJEnv{}, stubPMJExec{})
	require.NoError(t, err)
	require.Equal(t, "v", session["k"])
}

func TestRunPMJScript_pmAliasTestPass(t *testing.T) {
	art, err := RunPMJScript(context.Background(), true, `
pm.test('ok', () => {});
`, time.Second,
		nil, nil, map[string]string{}, noopPMJEnv{}, stubPMJExec{})
	require.NoError(t, err)
	require.Len(t, art.Tests, 1)
	require.True(t, art.Tests[0].Passed)
	require.Equal(t, "ok", art.Tests[0].Name)
}

func TestRunPMJScript_expectFailureRecordsTest(t *testing.T) {
	_, err := RunPMJScript(context.Background(), false, `
pmj.expect(1).to.equal(2);
`, time.Second,
		nil,
		&entity.HTTPExecuteResult{StatusCode: 200},
		map[string]string{}, noopPMJEnv{}, stubPMJExec{})
	require.Error(t, err)
}

func TestRunPMJScript_timeoutInterrupt(t *testing.T) {
	src := `
for (;;) {}
`
	_, err := RunPMJScript(context.Background(), true, strings.TrimSpace(src), 80*time.Millisecond,
		nil, nil, map[string]string{}, noopPMJEnv{}, stubPMJExec{})
	require.Error(t, err)
	low := strings.ToLower(err.Error())
	require.True(t, strings.Contains(low, "interrupt") || strings.Contains(low, "timeout"), "got %v", err)
}
