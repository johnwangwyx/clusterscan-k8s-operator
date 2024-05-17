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
