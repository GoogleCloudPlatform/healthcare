"""Utility functions for the project deployment scripts."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import collections
import logging
import os
import string
import subprocess
import sys
import tempfile

import jsonschema
import yaml

# Options for running GCloud and shell commands in this module.
GcloudOptions = collections.namedtuple('GcloudOptions', [
    'dry_run',     # If True, no gcloud commands will be executed.
    'gcloud_bin',  # Location of the gcloud binary.
    ])
GCLOUD_OPTIONS = GcloudOptions(dry_run=True, gcloud_bin='gcloud')

# Schema file for project configuration YAML files.
_PROJECT_CONFIG_SCHEMA = 'project_config.yaml.schema'


def wait_for_yes_no(text):
  """Prompt user for Yes/No and return true if Yes/Y. Default to No."""
  if GCLOUD_OPTIONS.dry_run:
    return True

  while True:
    # For compatibility with both Python 2 and 3.
    if sys.version_info[0] < 3:
      prompt = raw_input(text)
    else:
      prompt = input(text)

    if not prompt or prompt[0] in 'nN':
      # Default to No.
      return False
    if prompt[0] in 'yY':
      return True
    # Not Y or N, Keep trying.


def read_yaml_file(path):
  """Reads and parses a YAML file.

  Args:
    path (string): The path to the YAML file.

  Returns:
    A dict holding the parsed contents of the YAML file, or None if the file
    could not be read or parsed.
  """
  try:
    with open(path, 'r') as stream:
      return yaml.load(stream)
  except (yaml.YAMLError, IOError) as e:
    logging.error('Error reading YAML file: %s', e)
    return None


def write_yaml_file(contents, path):
  """Saves a dictionary as a YAML file.

  Args:
    contents (dict): The contents to write to the YAML file.
    path (string): The path to the YAML file.
  """
  if GCLOUD_OPTIONS.dry_run:
    # If using dry_run mode, don't create the file, just print the contents.
    print('Contents of {}:'.format(path))
    print('===================================================================')
    print(yaml.safe_dump(contents, default_flow_style=False))
    print('===================================================================')
    return
  with open(path, 'w') as outfile:
    yaml.safe_dump(contents, outfile, default_flow_style=False)


def validate_config_yaml(config):
  """Validates a Project config YAML against the schema.

  Args:
    config (dict): The parsed contents of the project config YAML file.

  Raises:
    jsonschema.exceptions.ValidationError: if the YAML contents do not match the
      schema.
  """
  schema_file_path = os.path.join(
      os.path.dirname(os.path.realpath(sys.argv[0])), _PROJECT_CONFIG_SCHEMA)
  schema = read_yaml_file(schema_file_path)

  jsonschema.validate(config, schema)


class GcloudRuntimeError(Exception):
  """Runtime exception raised when gcloud return code is non-zero."""


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
    GcloudRuntimeError: when command execution returns a non-zero return code.
  """
  cmd = [GCLOUD_OPTIONS.gcloud_bin] + cmd
  if project_id is not None:
    cmd.extend(['--project', project_id])
  logging.info('Executing command: %s', ' '.join(cmd))
  if GCLOUD_OPTIONS.dry_run:
    # Don't run the command, just return a place-holder value
    print('>>>> {}'.format(' '.join(cmd)))
    return '__DRY_RUN_MODE__:__DRY_RUN_MODE__'
  p = subprocess.Popen(cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                       stderr=subprocess.PIPE, shell=False, bufsize=-1,
                       close_fds=True)
  out, err = p.communicate()
  logging.info('Command returned stdout: %s', out)
  if err:
    logging.info('Command returned stderr: %s', err)
  if p.returncode != 0:
    raise GcloudRuntimeError(
        'Command %s returned non-zero exit code: %s' % (cmd, p.returncode))
  return out.decode()


def create_new_deployment(deployment_template, deployment_name, project_id):
  """Creates a new Deployment Manager deployment from a template.

  Args:
    deployment_template (dict): The dictionary representation of a deployment
      manager YAML template.
    deployment_name (string): The name for the deployment.
    project_id (string): The project under which to create the deployment.
  """
  # Save the deployment manager template to a temporary file in the same
  # directory as the deployment manager templates.
  dm_template_dir = os.path.join(
      os.path.dirname(os.path.realpath(sys.argv[0])), 'templates')
  dm_template_file = tempfile.NamedTemporaryFile(suffix='.yaml',
                                                 dir=dm_template_dir)
  write_yaml_file(deployment_template, dm_template_file.name)

  # Create the deployment.
  run_gcloud_command(['deployment-manager', 'deployments', 'create',
                      deployment_name,
                      '--config', dm_template_file.name,
                      '--automatic-rollback-on-error'],
                     project_id)

  # Check deployment exists (and wasn't automcatically rolled back
  run_gcloud_command(['deployment-manager', 'deployments', 'describe',
                      deployment_name], project_id)


def create_notification_channel(alert_email, project_id):
  """Creates a new Stackdriver email notification channel.

  Args:
    alert_email (string): The email address to send alerts to.
    project_id (string): The project under which to create the channel.
  Returns:
    A string, the name of the notification channel
  Raises:
    GcloudRuntimeError: when the channel cannot be created.
  """
  # Create a config file for the new Email notification channel.
  config_file = tempfile.NamedTemporaryFile(suffix='.yaml')
  channel_config = {
      'type': 'email',
      'displayName': 'Email',
      'labels': {
          'email_address': alert_email
      }
  }
  write_yaml_file(channel_config, config_file.name)

  # Create the new channel and get its name.
  channel_name = run_gcloud_command(
      ['alpha', 'monitoring', 'channels', 'create',
       '--channel-content-from-file', config_file.name,
       '--format', 'value(name)'], project_id).strip()
  return channel_name


def create_alert_policy(
    resource_type, metric_name, policy_name, description, channel, project_id):
  """Creates a new Stackdriver alert policy for a logs-based metric.

  Args:
    resource_type (string): The resource type for the metric.
    metric_name (string): The name of the logs-based metric.
    policy_name (string): The name for the newly created alert policy.
    description (string): A description of the alert policy.
    channel (string): The Stackdriver notification channel to send alerts on.
    project_id (string): The project under which to create the alert.
  Raises:
    GcloudRuntimeError: when command execution returns a non-zero return code.
  """
  # Create a config file for the new alert policy.
  config_file = tempfile.NamedTemporaryFile(suffix='.yaml')
  # Send an alert if the metric goes above zero.
  alert_config = {
      'displayName': policy_name,
      'documentation': {
          'content': description,
          'mimeType': 'text/markdown',
      },
      'conditions': [{
          'conditionThreshold': {
              'comparison': 'COMPARISON_GT',
              'thresholdValue': 0,
              'filter': ('resource.type="{}" AND '
                         'metric.type="logging.googleapis.com/user/{}"'.format(
                             resource_type, metric_name)),
              'duration': '0s'
          },
          'displayName': 'No tolerance on {}!'.format(metric_name),
      }],
      'combiner': 'AND',
      'enabled': True,
      'notificationChannels': [channel],
  }
  write_yaml_file(alert_config, config_file.name)

  # Create the new alert policy.
  run_gcloud_command(['alpha', 'monitoring', 'policies', 'create',
                      '--policy-from-file', config_file.name], project_id)


def get_gcloud_user():
  """Returns the active authenticated gcloud account."""
  return run_gcloud_command(
      ['config', 'list', 'account', '--format', 'value(core.account)'],
      project_id=None).strip()


def get_deployment_manager_service_account(project_id):
  """Returns the deployment manager service account for the given project."""
  project_num = run_gcloud_command([
      'projects', 'describe', project_id,
      '--format', 'value(projectNumber)'], project_id=None).strip()
  return 'serviceAccount:{}@cloudservices.gserviceaccount.com'.format(
      project_num)


def get_log_sink_service_account(log_sink_name, project_id):
  """Gets the service account name for the given log sink."""
  sink_service_account = run_gcloud_command([
      'logging', 'sinks', 'describe', log_sink_name,
      '--format', 'value(writerIdentity)'], project_id).strip()
  # The name returned has a 'serviceAccount:' prefix, so remove this.
  return sink_service_account.split(':')[1]


def resolve_env_vars(config):
  """Recursively resolves environment variables in config values."""
  if isinstance(config, str):
    return string.Template(config).substitute(os.environ)
  elif isinstance(config, dict):
    return {k: resolve_env_vars(v) for k, v in config.items()}
  elif isinstance(config, list):
    return [resolve_env_vars(i) for i in config]
  else:
    return config
