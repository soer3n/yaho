package helm

import (
	"github.com/soer3n/yaho/internal/helm"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
)

// GetTestFilterSpecs returns testcases for testing filtering of values
func GetTestFilterSpecs() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			ReturnError: nil,
			ReturnValue: []*helm.ValuesRef{
				{
					Parent: "parent",
				},
			},
			Input: []*helm.ValuesRef{
				{
					Parent: "parent",
				},
			},
		},
	}
}
