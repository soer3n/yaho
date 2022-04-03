package helm

type repositoryMock struct {
	Name      string
	Namespace string
	IsPresent bool
	Charts    []chartMock
	Labels    map[string]string
	Auth      *credentialsMock
	URL       string
}

type credentialsMock struct {
	Password string
	User     string
}

type chartMock struct {
	Name       string
	Namespace  string
	Repository string
	Group      *string
	Versions   []chartVersionMock
	IsPresent  bool
	Labels     map[string]string
}

type chartVersionMock struct {
	Chart        string
	Namespace    string
	Version      string
	Group        *string
	Dependencies []chartVersionMock
	IsPresent    bool
	Path         string
	URL          string
	Auth         *credentialsMock
}

type valueRefMock struct {
	Key  string
	Mock valueMock
}

type valueMock struct {
	Name      string
	Namespace string
	Releases  []string
	IsPresent bool
	Values    map[string]interface{}
	Refs      []valueRefMock
}
