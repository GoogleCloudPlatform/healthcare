"""Test rule that fails if a project config is not valid."""

def _impl(ctx):
    """Core implementation of _config_test rule."""
    if ctx.attr.projects:
        content = "{apply} --dry_run --config_path={config_path} --enable_terraform={enable_terraform} --projects={projects} --terraform_configs_dir=$TEST_UNDECLARED_OUTPUTS_DIR".format(
            apply = ctx.executable._apply.short_path,
            config_path = ctx.file.config.short_path,
            enable_terraform = ctx.attr.enable_terraform,
            projects = ",".join(ctx.attr.projects),
        )
    else:
        content = "{apply} --dry_run --config_path={config_path} --enable_terraform={enable_terraform} --terraform_configs_dir=$TEST_UNDECLARED_OUTPUTS_DIR".format(
            apply = ctx.executable._apply.short_path,
            config_path = ctx.file.config.short_path,
            enable_terraform = ctx.attr.enable_terraform,
        )

    ctx.actions.write(
        content = content,
        output = ctx.outputs.executable,
        is_executable = True,
    )

    runfiles = [
        ctx.file._apply,
        ctx.file._generated_fields_schema,
        ctx.file._project_config_schema,
        ctx.file.config,
    ] + ctx.files.deps

    return [DefaultInfo(runfiles = ctx.runfiles(files = runfiles))]

_config_test = rule(
    attrs = {
        "config": attr.label(
            mandatory = True,
            doc = "The project config to validate.",
            allow_single_file = True,
        ),
        "deps": attr.label_list(
            allow_files = True,
            doc = "Additional dependent configs, templates or generated fields file to import.",
        ),
        "enable_terraform": attr.bool(
            default = False,
            doc = "Whether to enable Terraform.",
        ),
        "projects": attr.string_list(
            mandatory = False,
            doc = "Comma separated project IDs to test, or leave unspecified to test all projects.",
        ),
        "_apply": attr.label(
            default = Label("//cmd/apply"),
            doc = "The config loader binary. Internal attribute and should not be set by users.",
            cfg = "host",
            executable = True,
            allow_single_file = True,
        ),
        # The following attributes are added purely for the purpose of making
        # schema files available in the runfiles tree.
        "_generated_fields_schema": attr.label(
            default = Label("//:generated_fields.yaml.schema"),
            doc = "The generated fields schema. Internal attribute and should not be set by users.",
            cfg = "host",
            allow_single_file = True,
        ),
        "_project_config_schema": attr.label(
            default = Label("//:project_config.yaml.schema"),
            doc = "The project config schema. Internal attribute and should not be set by users.",
            cfg = "host",
            allow_single_file = True,
        ),
    },
    test = True,
    implementation = _impl,
)

def config_test(**kwargs):
    """Test rule that fails if a project config is not valid.

    Args:
      **kwargs: Same attrs of _config_test.
    """
    if "deps" in kwargs and kwargs["deps"] == []:
        fail("`deps` is specified but resolved to no file. It might be because your glob() pattern contains typos.")

    _config_test(**kwargs)
