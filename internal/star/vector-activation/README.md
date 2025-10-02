[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/landscape-vector-activation-controller)](https://api.reuse.software/info/github.com/konfidence-project/landscape-vector-activation-controller)

# landscape-vector-activation-controller

## About this project

The Vector Activation Controller

## Requirements and Setup

### Private Repository Dependencies
The  vector activation CRD is located in a separate (private) [repository](https://github.com/konfidence-project/crds). Make sure you have setup the ```GOPRIVATE``` env variable in order to access other project-konfidence repositories (```export GOPRIVATE=github.com/konfidence-project/*```). Alternatively, you can also [setup a work.go](https://go.dev/doc/tutorial/workspaces) file to manage dependencies locally. 

### Run locally 

Make sure you have a cluster running locally. You can create a kind cluster with ``` make setup-test-e2e ```. 


To install the crds, navigate to the location of the crds module and run:

```
make install
```

Run the controller locally: 

```
make run
```

### Debug locally
With the setup above, just attach a debugger to the go process. This should be supported by your ide. 

Apply a sample CR: 

```
kubectl apply -f path-to-cr-sample
```

Inspect the cluster: 

```
kubectl get vectoractivation 
kubectl describe vectoractivation my-cr-instance
```

Delete the CR: 

```
kubectl delete vectoractivation my-cr-instance
```

### Unit Tests
You can run the tests with:

```
make test
```


## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/konfidence-project/<your-project>/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/konfidence-project/<your-project>/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/konfidence-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2025 SAP SE or an SAP affiliate company and landscape-vector-activation-controller contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/konfidence-project/<your-project>).
