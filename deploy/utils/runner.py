"""Runner provides utilities to run functions.

It is useful for providing a global way to run any mutating function for dry
runs.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import subprocess

from absl import flags
from absl import logging

FLAGS = flags.FLAGS

flags.DEFINE_bool('dry_run', True,
                  ('By default, no gcloud commands will be executed. '
                   'Use --nodry_run to execute commands.'))


def run(f, *args, **kwargs):
  """Runs the given function if dry run is false, else prints and returns."""
  if FLAGS.dry_run:
    # TODO: pass this through depenency injection so we don't need
    # dry run checks in the code.
    return fake_run(f, *args, **kwargs).encode()
  return f(*args, **kwargs)


def run_command(cmd, get_output=False):
  """Runs the given command."""
  logging.info('Executing command: %s', ' '.join(cmd))
  if get_output:
    return run(subprocess.check_output, cmd).decode()
  else:
    run(subprocess.check_call, cmd)


def run_gcloud_command(cmd, project_id):
  """Execute a gcloud command and return the output.

  Args:
    cmd (list): a list of strings representing the gcloud command to run
    project_id (string): append `--project {project_id}` to the command. Most
      commands should specify the project ID, for those that don't, explicitly
      set this to None.

  Returns:
    A string, the output from the command execution.

  Raises:
    CalledProcessError: when command execution returns a non-zero return code.
  """
  gcloud_cmd = ['gcloud'] + cmd
  if project_id:
    gcloud_cmd.extend(['--project', project_id])
  return run_command(gcloud_cmd, get_output=True).strip()


def fake_run(f, *args, **kwargs):
  """Fake run handles fake commands so they don't need to pollute prod code."""
  call = '__DRY_RUN_CALL__: {}('.format(f.__name__)
  if args:
    call += str(*args)
  if kwargs:
    call += str(**kwargs)
  call += ')'
  logging.info(call)

  if f != subprocess.check_output:
    return call

  cmd = args[0]
  if cmd[0].endswith('load_config'):
    return subprocess.check_output(cmd).decode()
  elif cmd[:5] == ['gcloud', 'alpha', 'monitoring', 'channels', 'list']:
    return '__DRY_RUN_CHANNEL DRY_RUN@domain.com'
  elif cmd[:4] == ['gcloud', 'compute', 'instances', 'list']:
    return '__DRY_RUN_NAME__ __DRY_RUN_ID__'
  elif cmd[:6] == [
      'gcloud', 'deployment-manager', 'deployments', 'list', '--format', 'json'
  ]:
    return '{}'
  elif cmd[:3] == ['gcloud', 'projects', 'get-iam-policy']:
    return '{ bindings: []}'
  elif cmd[:2] == ['gsutil', 'ls']:
    return 'gs://forseti-server-dry-run'
  else:
    return call
