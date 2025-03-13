package e2e_tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestComplexStepExample(t *testing.T) {
	yaml := `
spec: {}
---
run:
  - name: greet_steppy
    step: ./steps/greeting
    inputs:
      name: steppy
      hungry: true
      favorites:
        foods: [hamburger]
  - name: greet_the_crew
    step: ./steps/crew
    inputs: {}
  - name: greet_joe
    step: ./steps/greeting
    inputs:
      name: joe
      age: 42
      favorites:
        characters: 
          - ${{steps.greet_the_crew.outputs.crew_name_1}}
          - ${{steps.greet_the_crew.outputs.crew_name_2}}
`
	expectedLog := `Running step "greet_steppy"
meet steppy who is 1 likes {"foods":["hamburger"]} and is hungry true
Running step "greet_sponge_bob"
meet sponge bob who is 5 likes {"pants":"square"} and is hungry false
Running step "greet_patrick_star"
meet patrick star who is 7 likes {"color":"red"} and is hungry true
Running step "greet_joe"
meet joe who is 42 likes {"characters":["sponge bob","patrick star"]} and is hungry false
`
	result, logs, err := testutil.StepRunner(t).Run(yaml)
	require.NoError(t, err)
	require.Contains(t, logs, expectedLog)
	require.Len(t, result.SubStepResults, 3)
}
