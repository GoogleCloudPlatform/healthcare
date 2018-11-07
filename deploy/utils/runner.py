"""Runner provides utilities to run functions.

It is useful for providing a global way to run any mutating function for dry
runs.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import logging
import subprocess

DRY_RUN = False


def run(f, *args, **kwargs):
  """Runs the given function if dry run is false, else prints and returns."""
  if not DRY_RUN:
    return f(*args, **kwargs)

  call = f.__name__ + '('
  if args:
    call += str(*args)
  if kwargs:
    call += str(**kwargs)
  call += ')'
  logging.info('DRY_RUN: ' + call)


def run_command(cmd, wait_for_output=True):
  """Runs the given command."""
  logging.info('Executing command: ' + ' '.join(cmd))
  fn = subprocess.check_output if wait_for_output else subprocess.check_call
  return run(fn, cmd)
