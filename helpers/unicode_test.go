package helpers_test

import (
	"fmt"
	"isp-gate-service/helpers"
	"os"
	"testing"

	"github.com/txix-open/isp-kit/json"

	"github.com/stretchr/testify/require"
)

func TestForceUnescapingUnicode(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Raw      string
		Expected string
	}

	data, err := os.ReadFile("./test_data/json_cases.json")
	require.NoError(t, err)

	var testCases []testCase
	require.NoError(t, json.Unmarshal(data, &testCases))

	for i, tt := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			t.Parallel()

			out := helpers.UnescapeUnicode([]byte(tt.Raw))
			require.JSONEq(t, tt.Expected, string(out))

			jsonOut := helpers.UnescapeUnicodeJson([]byte(tt.Raw))
			require.JSONEq(t, tt.Expected, string(jsonOut))
		})
	}
}
