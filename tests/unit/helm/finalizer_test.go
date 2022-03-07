package helm

/*
func TestFinalizerHandleRelease(t *testing.T) {
	clientMock, httpMock := helmmocks.GetFinalizerMock()
	assert := assert.New(t)

	settings := utils.GetEnvSettings(map[string]string{})
	testObj, _ := release.New(testcases.GetTestFinalizerRelease(), settings, logf.Log, clientMock, httpMock, kube.Client{})
	testObj.Config = testcases.GetTestFinalizerFakeActionConfig(t)

	if err := testObj.Config.Releases.Create(testcases.GetTestFinalizerDeployedReleaseObj()); err != nil {
		log.
			Print(err)
	}

	for _, v := range testcases.GetTestFinalizerSpecsRelease() {

		err := testObj.RemoveRelease()
		assert.Equal(v.ReturnError, err)
	}
}
*/
