# GDPR-Aligned Data Warehouse DPT Templates

**Status:** Stable

**Solution Guide:**
[Using Data Protection Toolkit as part of a GDPR Compliance Strategy](http://services.google.com/fh/files/misc/gdpr_technical_solution_guide.pdf)

The essential templates required to deploy the GDPR-aligned datawarehouse are:

- `variables.yaml`
- `config.yaml`

## Deploying the templates

`config.yaml` file is imported by `variables.yaml` file. So the entry point
would be `variables.yaml` file.

```shell
  bazel run //cmd/apply:apply -- --config_path=./variables.yaml
```

The setup steps for the DPT can be found [here](../..).
