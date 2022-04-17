[![codecov](https://codecov.io/gh/soer3n/yaho/branch/master/graph/badge.svg?token=DCPVNPSIFF)](https://codecov.io/gh/soer3n/yaho)
[![Go Report Card](https://goreportcard.com/badge/soer3n/yaho)](https://goreportcard.com/report/soer3n/yaho)

# Yet Another Helm Operator 

This operator is for managing helm repositories, releases and values in a declarative way. This project tries to picture helm as an kubernetes api extension. Through a custom resource for values reusing of them in different releases with same sub specifications is one feature. Another is to use kubernetes rbac for restricting helm usage for specific cluster configs. And there are no local files which could differ in multiple workstations.

## Docs

Documentation is currently work in progress due to a migration to hugo generated content.

## Contributing

Pull requests are welcome. As these project is in a very early stage there is currently no traditional contribution guideline due to the fact that actually every issue is a bigger change which can bring incompatibility on update processes of this operator.

But everyone can feel welcome to mention ideas and adding features which makes sense what could be actually everything what you can do with helm. More than view is needed for a proper further development.

## License
[LICENSE](LICENSE)
