package helm

/*
func TestChartAddOrUpdateMap(t *testing.T) {
	settings := cli.New()
	cases := testcases.GetTestHelmChartMaps()
	mapChannel := make(chan *helmv1alpha1.Chart, len(cases))
	returnChannel := make(chan error, len(cases))
	clientMock, httpMock := helmmocks.GetChartMock()
	var err error
	mu := &sync.Mutex{}

	assert := assert.New(t)

	for _, v := range cases {
		obj := v.Input.(*helmv1alpha1.Chart)
		go func() {
			defer mu.Unlock()
			mu.Lock()
			for _, i := range testcases.GetTestRepoChartVersions() {
				ver := i.Input.([]*repo.ChartVersion)
				testObj := chart.New(obj.ObjectMeta.Name, "https://foo.bar/charts/foo-0.0.1.tgz", ver, settings, logf.Log, "test", clientMock, httpMock, kube.Client{})
				// rel, _ := v.Input.(map[string]*helmv1alpha1.Chart)

				err = testObj.AddOrUpdateChartMap(testcases.GetTestChartRepo(), mapChannel)
				returnChannel <- err
			}
		}()

		go func() {
			for i := range mapChannel {
				fmt.Printf("receive channel event %v", i)
			}
		}()

		go func(v types.TestCase) {
			for i := range returnChannel {
				fmt.Printf("receive return value %v", i)
				assert.Equal(v.ReturnError, i)
			}
		}(v)
	}
}
*/
