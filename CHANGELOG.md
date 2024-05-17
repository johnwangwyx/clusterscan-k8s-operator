## ADDED
Add parameters for `ClusterScanSpec` and `ClusterScanStatus`.
Add custom reconciliation logic in `internal/controller/clusterscan_controller.go`
Add `config/rbac/clusterrolebinding.yaml` and `config/rbac/clusterrole.yaml` to create access roles for the operator to monitor and create jobs and cronjobs.
Add `job_image/ansible-demo` ansible toy example for demo purposes.
Add `config/samples/johnwangwyx.io_v1_clusterscan_ansible_demo.yaml` sample CRD.

## CHANGED
change `LastExecutionTime` to `LastStatusChange` to allow finer timing management.

## DEBUGGED
