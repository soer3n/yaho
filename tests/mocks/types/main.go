package types

// TestCase represents a struct with setup structs, expected return structs and errors of tested funcs
type TestCase struct {
	ReturnValue interface{}
	ReturnError error
	Input       interface{}
}
