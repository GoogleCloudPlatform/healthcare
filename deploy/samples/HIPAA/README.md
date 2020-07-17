# HIPAA-Aligned Analytics and AI/ML Platform DPT Templates

**Status:** Stable

**Solution Guide:**
[Setting up a HIPAA-Aligned workload using Data Protection Toolkit](http://services.google.com/fh/files/misc/hipaa_technical_solution_guide.pdf)

The essential templates required to deploy the HIPAA-Aligned Analytics and AI/ML
Platform are:

- `variables.yaml`
- `config.yaml`

## Deploying the templates

`config.yaml` file is imported by `variables.yaml` file. So the entry point
would be `variables.yaml` file.

```shell
  bazel run //cmd/apply:apply -- --config_path=./variables.yaml
```

The setup steps for the DPT can be found [here](../..).
