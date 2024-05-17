# DEMO
The demo Video can be found on: https://youtu.be/jq9gt4NBuGs

# clusterscan-k8s-operator
A clusterscan operator that can execute configurable jobs and cronjobs.

# Changes
You can use this link to see the changes added after the kubebuilder init:
https://github.com/johnwangwyx/clusterscan-k8s-operator/compare/ba7959014ec3aa5524f959c9aa8f2f5b70bccb4a..95a2678a5e3b87e81c759a19f0a4a493f460b741

# Future considerations:
* Adding a figuration management file or system logic to manage the variables and job-specific config.
* Implement the logic for filtering for `Target` and checking for concurrent operation when `AllowConcurrency` spec is set to False.
* Adding a lastOp field to the `Status` to track of the last operation that the underlining job performed.
* Set ownership for the jobs created by the cronjob to the same operator. With testing, it is found that the cron children's job will not inherit this ownership and the cluster can miss reconciliation for them. (The workaround is to have a periodic reconciliation but this sort of hinders the event-based approach we had here)
* Introduce more scheduling options beyond standard cron expressions(like dynamic scheduling based on cluster events or metrics thresholds that I am thinking of)
* Ensure that all resources created by the operator are properly cleaned up when the operator itself is deleted.

## Getting Started

### Prerequisites
- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/clusterscan-k8s-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/clusterscan-k8s-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/clusterscan-k8s-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/clusterscan-k8s-operator/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

