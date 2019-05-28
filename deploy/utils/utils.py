"""Utility functions for the project deployment scripts."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import glob
import json
import os
import string
import sys
import tempfile

from absl import flags

import jsonschema
import yaml

from deploy.utils import runner

FLAGS = flags.FLAGS

# Schema file for project configuration YAML files.
_PROJECT_CONFIG_SCHEMA = os.path.join(
    os.path.dirname(__file__), '../project_config.yaml.schema')

# Retrieve all files matching the pattern and add them into import_files.
IMPORT_PATTERN_TAG = 'imports'
# Merge the files in import_files into the dict where it is declared.
IMPORT_FILES_TAG = 'import_files'


def normalize_path(path):
  """Normalizes paths specified through a local run or Bazel invocation."""
  path = os.path.expandvars(os.path.expanduser(path))
  if path.startswith('gs://') or os.path.isabs(path):
    return path

  # Path is relative from where the script was launched from.
  # When using `bazel run`, the environment variable BUILD_WORKING_DIRECTORY
  # will be set to the path where the command was run from.
  cwd = os.environ.get('BUILD_WORKING_DIRECTORY', os.getcwd())
  return os.path.abspath(os.path.join(cwd, path))


def wait_for_yes_no(text):
  """Prompt user for Yes/No and return true if Yes/Y. Default to No."""
  if FLAGS.dry_run:
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
  with open(path, 'r') as stream:
    return yaml.load(stream)


def write_yaml_file(contents, path):
  """Saves a dictionary as a YAML file.

  Args:
    contents (dict): The contents to write to the YAML file.
    path (string): The path to the YAML file.
  """
  yaml.Dumper.ignore_aliases = lambda *args: True

  if FLAGS.dry_run:
    # If using dry_run mode, don't create the file, just print the contents.
    print('Contents of {}:'.format(path))
    print('===================================================================')
    yaml.dump(contents, sys.stdout)
    print('===================================================================')
    return

  with open(path, 'w') as outfile:
    yaml.dump(contents, outfile, default_flow_style=False)


def validate_config_yaml(config):
  """Validates a Project config YAML against the schema.

  Args:
    config (dict): The parsed contents of the project config YAML file.

  Raises:
    jsonschema.exceptions.ValidationError: if the YAML contents do not match the
      schema.
  """
  schema = read_yaml_file(_PROJECT_CONFIG_SCHEMA)

  jsonschema.validate(config, schema)


def run_deployment(deployment_template, deployment_name, project_id):
  """Creates a new Deployment Manager deployment from a template.

  Args:
    deployment_template (dict): The dictionary representation of a deployment
      manager YAML template.
    deployment_name (string): The name for the deployment.
    project_id (string): The project under which to create the deployment.
  """
  # Save the deployment manager template to a temporary file in the same
  # directory as the deployment manager templates.
  dm_template_file = tempfile.NamedTemporaryFile(suffix='.yaml')
  write_yaml_file(deployment_template, dm_template_file.name)

  if deployment_exists(deployment_name, project_id):
    gcloud_cmd = [
        'deployment-manager',
        'deployments',
        'update',
        deployment_name,
        '--config',
        dm_template_file.name,
        '--delete-policy',
        'ABANDON',
    ]
  else:
    gcloud_cmd = [
        'deployment-manager',
        'deployments',
        'create',
        deployment_name,
        '--config',
        dm_template_file.name,
        '--automatic-rollback-on-error',
    ]

  # Create the deployment.
  runner.run_gcloud_command(gcloud_cmd, project_id=project_id)

  # Check deployment exists (and wasn't automcatically rolled back)
  runner.run_gcloud_command(
      ['deployment-manager', 'deployments', 'describe', deployment_name],
      project_id=project_id)


def deployment_exists(deployment_name, project_id):
  """Determine whether the deployment exists.

  Args:
    deployment_name (string): name of deployment.
    project_id: ID of project.

  Returns:
    bool: True if deployment exists in the projet.
  """
  out = runner.run_gcloud_command(
      ['deployment-manager', 'deployments', 'list', '--format', 'json'],
      project_id=project_id)

  for info in json.loads(out):
    if deployment_name == info['name']:
      return True
  return False


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
  channel_name = runner.run_gcloud_command(
      ['alpha', 'monitoring', 'channels', 'create',
       '--channel-content-from-file', config_file.name,
       '--format', 'value(name)'],
      project_id=project_id).strip()
  return channel_name


def create_alert_policy(
    resource_types, metric_name, policy_name, description, channel, project_id):
  """Creates a new Stackdriver alert policy for a logs-based metric.

  Args:
    resource_types (list[str]): A list of resource types for the metric.
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

  resource_type_str = ''
  if len(resource_types) > 1:
    index = 0
    resource_type_str = 'one_of(\"'+resource_types[index]+'\"'
    while index < len(resource_types) - 1:
      index += 1
      resource_type_str += ',\"'+resource_types[index]+'\"'
    resource_type_str = resource_type_str + ')'
  else:
    resource_type_str = '\"' + resource_types[0] + '\"'

  alert_filter = ('resource.type={} AND '
                  'metric.type="logging.googleapis.com/user/{}"').format(
                      resource_type_str, metric_name)

  condition_threshold = {
      'comparison': 'COMPARISON_GT',
      'thresholdValue': 0,
      'filter': alert_filter,
      'duration': '0s'
  }

  conditions = [{'conditionThreshold': condition_threshold,
                 'displayName': 'No tolerance on {}!'.format(
                     metric_name)}]

  # Send an alert if the metric goes above zero.
  alert_config = {
      'displayName': policy_name,
      'documentation': {
          'content': description,
          'mimeType': 'text/markdown',
      },
      'conditions': conditions,
      'combiner': 'AND',
      'enabled': True,
      'notificationChannels': [channel],
  }
  write_yaml_file(alert_config, config_file.name)

  # Create the new alert policy.
  runner.run_gcloud_command(
      ['alpha', 'monitoring', 'policies', 'create',
       '--policy-from-file', config_file.name],
      project_id=project_id)


def get_gcloud_user():
  """Returns the active authenticated gcloud account."""
  return runner.run_gcloud_command(
      ['config', 'list', 'account', '--format', 'value(core.account)'],
      project_id=None).strip()


def get_project_number(project_id):
  """Returns the project number the given project."""
  return runner.run_gcloud_command(
      ['projects', 'describe', project_id,
       '--format', 'value(projectNumber)'],
      project_id=None).strip()


def get_deployment_manager_service_account(project_id):
  """Returns the deployment manager service account for the given project."""
  return 'serviceAccount:{}@cloudservices.gserviceaccount.com'.format(
      get_project_number(project_id))


def get_log_sink_service_account(log_sink_name, project_id):
  """Gets the service account name for the given log sink."""
  sink_service_account = runner.run_gcloud_command([
      'logging', 'sinks', 'describe', log_sink_name,
      '--format', 'value(writerIdentity)'], project_id).strip()
  # The name returned has a 'serviceAccount:' prefix, so remove this.
  return sink_service_account.split(':')[1]


def get_gce_instance_info(project_id):
  """Gets a list of GCE instance info for each instance."""
  output = runner.run_gcloud_command(
      ['compute', 'instances', 'list', '--format', 'value(name,id)'],
      project_id=project_id)
  if not output:
    return []

  instance_info = []
  for line in output.split('\n'):
    name, instance_id = line.split()
    instance_info.append({'name': name, 'id': instance_id})
  return instance_info


def resolve_env_vars(config):
  """Recursively resolves (in place) environment variables in config strings.

  Args:
    config: dictionary or list from a parsed YAML file.
  """
  keys = []
  if isinstance(config, dict):
    keys = config.keys()
  elif isinstance(config, list):
    keys = range(len(config))
  else:
    return

  for k in keys:
    # Only do substitutions on strings.
    v = config[k]
    if isinstance(v, str):
      config[k] = string.Template(v).substitute(os.environ)
    else:
      # Recursively handle lists and dictionaries.
      resolve_env_vars(v)


def merge_dicts(dst, src):
  """Merge src into dst.

  Args:
    dst: Dictionary for all configs.
    src: Dictionary to be merged.

  Raises:
    TypeError: If two dictionaries have the same key and different value types.
  """
  for key, val in src.items():
    if key in dst:
      root_value = dst[key]
      if isinstance(val, list) and isinstance(root_value, list):
        root_value.extend(val)
      elif isinstance(val, dict) and isinstance(root_value, dict):
        merge_dicts(root_value, val)
      else:
        raise TypeError('Conflict key %s in config files.' % key)
    else:
      dst[key] = val


def get_import_files(overall, overall_path):
  """Get all imported file normalized paths.

  Args:
    overall: dictionary or list from a parsed YAML file.
    overall_path: The path of the overall YAML file.

  Returns:
    A list holding all the imported file normalized paths.
  """
  import_list = overall.get(IMPORT_FILES_TAG, [])
  import_files = set()
  for file_path in import_list:
    import_files.add(
        os.path.normpath(
            os.path.join(os.path.dirname(overall_path), file_path)))
  ret_list = list(
      import_files.union(set(expand_imports(overall, overall_path))))
  ret_list.sort()
  return ret_list


def expand_imports(overall, overall_path):
  """Find all to be imported files to extend import_files.

  Args:
    overall: dictionary or list from a parsed YAML file.
    overall_path: The path of the overall YAML file.

  Returns:
    A list holding all the imported file paths.
  """
  imports_patterns = overall.get(IMPORT_PATTERN_TAG, [])
  all_files = set()
  for pattern in imports_patterns:
    absolute_pattern = os.path.normpath(
        os.path.join(os.path.dirname(overall_path), pattern))
    for path in glob.glob(absolute_pattern):
      if path != overall_path:
        all_files.add(path)

  return all_files


def load_config(overall_path):
  """Reads and parses a YAML file.

  Args:
    overall_path (string): The path to the YAML file.

  Returns:
    A dict holding the parsed contents of the YAML file, or None if the file
    could not be read or parsed.
  """
  overall = read_yaml_file(overall_path)
  if not overall:
    return None
  import_files = get_import_files(overall, overall_path)
  for inc_file in import_files:
    inc_contents = read_yaml_file(inc_file)
    merge_dicts(overall, inc_contents)

  resolve_env_vars(overall)
  return overall


class InvalidConfigError(Exception):
  """The exception when the config file is invalid."""
  pass
