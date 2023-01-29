[![codecov](https://codecov.io/gh/soer3n/yaho/branch/master/graph/badge.svg?token=DCPVNPSIFF)](https://codecov.io/gh/soer3n/yaho)
[![Go Report Card](https://goreportcard.com/badge/soer3n/yaho)](https://goreportcard.com/report/soer3n/yaho)
![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/soer3n/yaho)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/soer3n/yaho)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/soer3n/yaho/ci.yaml?label=Tests&logo=Tests)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/soer3n/yaho/release.yaml?label=Release)
![GitHub](https://img.shields.io/github/license/soer3n/yaho)

# Yet Another Helm Operator 

This operator is for managing helm repositories, releases and values in a declarative way. This project tries to picture helm as an kubernetes api extension. Through a custom resource for values reusing of them in different releases with same sub specifications is one feature. Another is to use kubernetes rbac for restricting helm usage for specific cluster configs. And there are no local files which could differ on multiple workstations.

## Docs

Documentation is build with [hugo](https://github.com/gohugoio/hugo) and can be found [here](https://soer3n.github.io/yaho/).

## Contributing

Pull requests are welcome. As these project is in a early stage there is currently no guideline for it. 

If you find a bug while running the operator please open a [pull request](https://github.com/soer3n/yaho/pulls) or an [issue](https://github.com/soer3n/yaho/issues). If you want to make a feature request you should open an issue at first.

## License
The license file can be viewed [here](LICENSE).
