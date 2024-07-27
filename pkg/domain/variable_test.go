package domain

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stretchr/testify/require"
)

func TestVariable_Assign(t *testing.T) {
	tests := map[string]struct {
		value    *Value
		variable *Variable
		wantErr  string
	}{
		"cannot assign sensitive variable to non-sensitive variable": {
			value:    NewStringValue("new_value", true, "outputs.secret"),
			variable: NewVariable(structpb.NewStringValue("old_value"), false),
			wantErr:  `non-sensitive input cannot derive value using sensitive value(s) "outputs.secret"`,
		},
		"can assign sensitive variable to sensitive variable": {
			value:    NewStringValue("new_value", true, "outputs.secret"),
			variable: NewVariable(structpb.NewStringValue("old_value"), true),
		},
		"can assign non-sensitive variable to non-sensitive variable": {
			value:    NewStringValue("new_value", false, ""),
			variable: NewVariable(structpb.NewStringValue("old_value"), false),
		},
		"can assign non-sensitive variable to sensitive variable": {
			value:    NewStringValue("new_value", false, ""),
			variable: NewVariable(structpb.NewStringValue("old_value"), true),
		},
	}

	for _, test := range tests {
		err := test.variable.Assign(test.value)

		if test.wantErr == "" {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.Equal(t, test.wantErr, err.Error())
		}
	}
}
