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

DRY_RUN = True
flags.DEFINE_bool('dry_run', True,
                  ('By default, no gcloud commands will be executed. '
                   'Use --nodry_run to execute commands.'))
flags.DEFINE_string('gcloud_bin', 'gcloud',
                    'Location of the gcloud binary.')


def run(f, *args, **kwargs):
  """Runs the given function if dry run is false, else prints and returns."""
  if not FLAGS.dry_run:
    return f(*args, **kwargs)

  call = '__DRY_RUN_CALL__: {}('.format(f.__name__)
  if args:
    call += str(*args)
  if kwargs:
    call += str(**kwargs)
  call += ')'
  logging.info(call)
  return call.encode()


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
  gcloud_cmd = [FLAGS.gcloud_bin] + cmd
  if project_id:
    gcloud_cmd.extend(['--project', project_id])
  return run_command(gcloud_cmd, get_output=True).strip()
