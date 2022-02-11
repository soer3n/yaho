package helm

import (
	"github.com/soer3n/yaho/internal/values"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
)

// GetTestFilterSpecs returns testcases for testing filtering of values
func GetTestFilterSpecs() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			ReturnError: nil,
			ReturnValue: []*values.ValuesRef{
				{
					Parent: "parent",
				},
			},
			Input: []*values.ValuesRef{
				{
					Parent: "parent",
				},
			},
		},
	}
}
