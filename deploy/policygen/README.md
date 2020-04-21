# Policy Generator

Status: ALPHA

A security policy generator which generates policies for two purposes:

1.  Typical policies enforced in a HIPAA aligned GCP environment.
1.  Policies based on Terraform configs to monitor GCP changes that are not
    deployed by Terraform.

Currently supported Policy Libraries:

*   [Forseti Policy Library](https://github.com/forseti-security/policy-library)
    as YAML files.
*   [Google Cloud Platform Organization Policy Constraints](https://cloud.google.com/resource-manager/docs/organization-policy/org-policy-constraints)
    as Terraform configs.
