#!/usr/bin/python
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""A script to create a new dataset project.

Create a project config YAML file (see README.md for details) then run the
script with:
  ./create_project.py --project_yaml=my_project_config.yaml --nodry_run

To preview the commands that will run, use `--dry_run`.

If the script fails part way through, you can retry from the same step of the
failing project using: `--resume_from_project=project-id --resume_from_step=N`,
where project-id is the project and N is the step number that failed.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import collections
import logging

import jsonschema

import utils

# Name of the Log Sink created in the data_project deployment manager template.
_LOG_SINK_NAME = 'audit-logs-to-bigquery'

# Configuration for deploying a single project.
ProjectConfig = collections.namedtuple('ProjectConfig', [
    # Dictionary of configuration values for all projects.
    'overall',
    # Dictionary of configuration values for this project.
    'project',
    # Dictionary of configuration values of the remote audit logs project, or
    # None if the project uses local logs.
    'audit_logs_project',
    ])


def CreateNewProject(config):
  """Creates the new GCP project."""
  logging.info('Creating a new GCP project...')
  org_id = config.overall.get('organization_id')
  project_id = config.project['project_id']

  create_project_command = ['projects', 'create', project_id]
  if org_id:
    create_project_command.extend(['--organization', org_id])
  else:
    logging.info('Deploying without a parent organization.')
  # Create the new project.
  utils.RunGcloudCommand(create_project_command, project_id=None)


def SetupBilling(config):
  """Sets the billing account for this project."""
  logging.info('Setting up billing...')
  billing_acct = config.overall['billing_account']
  project_id = config.project['project_id']
  # Set the appropriate billing account for this project:
  utils.RunGcloudCommand(['beta', 'billing', 'projects', 'link', project_id,
                          '--billing-account', billing_acct],
                         project_id=None)


def EnableDeploymentManager(config):
  """Enables Deployment manager, with role/owners for its service account."""
  logging.info('Setting up Deployment Manager...')
  project_id = config.project['project_id']

  # Enabled Deployment Manger and Cloud Resource Manager for this project.
  utils.RunGcloudCommand(['services', 'enable', 'deploymentmanager',
                          'cloudresourcemanager.googleapis.com'],
                         project_id)

  # Grant deployment manager service account (temporary) owners access.
  dm_service_account = utils.GetDeploymentManagerServiceAccount(project_id)
  utils.RunGcloudCommand(['projects', 'add-iam-policy-binding', project_id,
                          '--member', dm_service_account,
                          '--role', 'roles/owner'],
                         project_id=None)


def DeployGcsAuditLogs(config):
  """Deploys the GCS logs bucket to the remote audit logs project, if used."""
  # The GCS logs bucket must be created before the data buckets.
  if not config.audit_logs_project:
    logging.info('Using local GCS audit logs.')
    return
  logs_gcs_bucket = config.project['audit_logs'].get('logs_gcs_bucket')
  if not logs_gcs_bucket:
    logging.info('No remote GCS logs bucket required.')
    return

  logging.info('Creating remote GCS logs bucket.')
  data_project_id = config.project['project_id']
  logs_project = config.audit_logs_project
  audit_project_id = logs_project['project_id']

  deployment_name = 'audit-logs-{}-gcs'.format(
      data_project_id.replace('_', '-'))
  dm_template_dict = {
      'imports': [{'path': 'remote_audit_logs.py'}],
      'resources': [{
          'type': 'remote_audit_logs.py',
          'name': deployment_name,
          'properties': {
              'owners_group': logs_project['owners_group'],
              'auditors_group': config.project['auditors_group'],
              'logs_gcs_bucket': logs_gcs_bucket,
          },
      }]
  }
  utils.CreateNewDeployment(dm_template_dict, deployment_name, audit_project_id)


def DeployProjectResources(config):
  """Deploys resources into the new data project."""
  logging.info('Deploying Project resources...')
  setup_account = utils.GetGcloudUser()
  has_organization = bool(config.overall.get('organization_id'))
  project_id = config.project['project_id']
  dm_service_account = utils.GetDeploymentManagerServiceAccount(project_id)

  # Build a deployment config for the data_project.py deployment manager
  # template.
  # Shallow copy is sufficient for this script.
  properties = config.project.copy()
  # Remove the current user as an owner of the project if project is part of an
  # organization.
  properties['has_organization'] = has_organization
  if has_organization:
    properties['remove_owner_user'] = setup_account

  # Change audit_logs to either local_audit_logs or remote_audit_logs in the
  # deployment manager template properties.
  audit_logs = properties.pop('audit_logs')
  if config.audit_logs_project:
    properties['remote_audit_logs'] = {
        'audit_logs_project_id': config.audit_logs_project['project_id'],
        'logs_bigquery_dataset_id': audit_logs['logs_bigquery_dataset']['name'],
    }
    # Logs GCS bucket is not required for projects without data GCS buckets.
    if 'logs_gcs_bucket' in audit_logs:
      properties['remote_audit_logs']['logs_gcs_bucket_name'] = (
          audit_logs['logs_gcs_bucket']['name'])
  else:
    properties['local_audit_logs'] = audit_logs
  dm_template_dict = {
      'imports': [{'path': 'data_project.py'}],
      'resources': [{
          'type': 'data_project.py',
          'name': 'data_project_deployment',
          'properties': properties,
      }]
  }

  # Create the deployment.
  utils.CreateNewDeployment(dm_template_dict, 'data-project-deployment',
                            project_id)

  # Remove Owners role from the DM service account.
  utils.RunGcloudCommand(['projects', 'remove-iam-policy-binding', project_id,
                          '--member', dm_service_account,
                          '--role', 'roles/owner'],
                         project_id=None)


def DeployBigQueryAuditLogs(config):
  """Deploys the BigQuery audit logs dataset, if used."""
  data_project_id = config.project['project_id']
  logs_dataset = config.project['audit_logs']['logs_bigquery_dataset'].copy()
  if config.audit_logs_project:
    logging.info('Creating remote BigQuery logs dataset.')
    audit_project_id = config.audit_logs_project['project_id']
    owners_group = config.audit_logs_project['owners_group']
  else:
    logging.info('Creating local BigQuery logs dataset.')
    audit_project_id = data_project_id
    logs_dataset['name'] = 'audit_logs'
    owners_group = config.project['owners_group']

  # Get the service account for the newly-created log sink.
  logs_dataset['log_sink_service_account'] = utils.GetLogSinkServiceAccount(
      _LOG_SINK_NAME, data_project_id)

  deployment_name = 'audit-logs-{}-bq'.format(
      data_project_id.replace('_', '-'))
  dm_template_dict = {
      'imports': [{'path': 'remote_audit_logs.py'}],
      'resources': [{
          'type': 'remote_audit_logs.py',
          'name': deployment_name,
          'properties': {
              'owners_group': owners_group,
              'auditors_group': config.project['auditors_group'],
              'logs_bigquery_dataset': logs_dataset,
          },
      }]
  }
  utils.CreateNewDeployment(dm_template_dict, deployment_name, audit_project_id)


def CreateStackdriverAccount(config):
  """Prompts the user to create a new Stackdriver Account."""
  # Creating a Stackdriver account cannot be done automatically, so ask the
  # user to create one.
  if 'stackdriver_alert_email' not in config.project:
    logging.warning('No Stackdriver alert email specified, skipping creation '
                    'of Stackdriver account.')
    return
  logging.info('Creating Stackdriver account.')
  project_id = config.project['project_id']

  message = """
  ------------------------------------------------------------------------------
  To create email alerts, this project needs a Stackdriver account.
  Create a new Stackdriver account for this project by visiting:
      https://console.cloud.google.com/monitoring?project={}

  Only add this project, and skip steps for adding additional GCP or AWS
  projects. You don't need to install Stackdriver Agents.

  IMPORTANT: Wait about 5 minutes for the account to be created.

  For more information, see: https://cloud.google.com/monitoring/accounts/

  After the account is created, enter [Y] to continue, or enter [N] to skip the
  creation of Stackdriver alerts.
  ------------------------------------------------------------------------------
  """.format(project_id)
  print(message)

  # Keep trying until Stackdriver account is ready, or user skips.
  while True:
    if not utils.WaitForYesNo('Account created [y/N]?'):
      logging.warning('Skipping creation of Stackdriver Account.')
      return

    # Verify account was created.
    try:
      utils.RunGcloudCommand(['alpha', 'monitoring', 'policies', 'list'],
                             project_id)
      return
    except utils.GcloudRuntimeError as e:
      logging.error('Error reading Stackdriver account %s', e)
      print('Could not find Stackdriver account.')


def CreateAlerts(config):
  """"Creates Stackdriver alerts for logs-based metrics."""
  # Stackdriver alerts can't yet be created in Deployment Manager, so create
  # them here.
  alert_email = config.project.get('stackdriver_alert_email')
  if alert_email is None:
    logging.warning('No Stackdriver alert email specified, skipping creation '
                    'of Stackdriver alerts.')
    return
  project_id = config.project['project_id']

  # Create an email notification channel for alerts.
  logging.info('Creating Stackdriver notification channel.')
  channel = utils.CreateNotificationChannel(alert_email, project_id)

  logging.info('Creating Stackdriver alerts.')
  utils.CreateAlertPolicy(
      'global', 'iam-policy-change-count', 'IAM Policy Change Alert',
      ('This policy ensures the designated user/group is notified when IAM '
       'policies are altered.'), channel, project_id)

  utils.CreateAlertPolicy(
      'gcs_bucket', 'bucket-permission-change-count',
      'Bucket Permission Change Alert',
      ('This policy ensures the designated user/group is notified when '
       'bucket/object permissions are altered.'), channel, project_id)

  for data_bucket in config.project.get('data_buckets', []):
    # Every bucket with 'expected_users' has an expected-access alert.
    if 'expected_users' in data_bucket:
      bucket_name = project_id + data_bucket['name_suffix']
      metric_name = 'unexpected-access-' + bucket_name
      utils.CreateAlertPolicy(
          'gcs_bucket', metric_name,
          'Unexpected Access to {} Alert'.format(bucket_name),
          ('This policy ensures the designated user/group is notified when '
           'bucket {} is accessed by an unexpected user.'.format(bucket_name)),
          channel, project_id)


# The steps to set up a project, so the script can be resumed part way through
# on error. Each is a function that takes a config dictionary and
# raises a GcloudRuntimeError on errors.
_SETUP_STEPS = [
    CreateNewProject,
    SetupBilling,
    EnableDeploymentManager,
    DeployGcsAuditLogs,
    DeployProjectResources,
    DeployBigQueryAuditLogs,
    CreateStackdriverAccount,
    CreateAlerts,
]


def SetupNewProject(config, starting_step):
  """Run the full process for initalizing a single new project.

  Args:
    config (ProjectConfig): The config of a single project to setup.
    starting_step (int): The step number (indexed from 1) in _SETUP_STEPS to
       begin from.

  Returns:
    A boolean, true if the project was deployed successfully, false otherwise.
  """
  total_steps = len(_SETUP_STEPS)
  for step_num in range(starting_step, total_steps + 1):
    logging.info('Step %s/%s', step_num, total_steps)
    try:
      _SETUP_STEPS[step_num - 1](config)
    except utils.GcloudRuntimeError as e:
      logging.error('Setup failed on step %s: %s', step_num, e)
      logging.error(
          'To continue the script, run with flags: --resume_from_project=%s '
          '--resume_from_step=%s', config.project['project_id'], step_num)
      return False

  logging.info('Setup completed successfully.')
  return True


def main(args):
  logging.basicConfig(level=getattr(logging, args.log))
  utils.GCLOUD_OPTIONS = utils.GcloudOptions(
      dry_run=args.dry_run, gcloud_bin=args.gcloud_bin)

  # Read and parse the project configuration YAML file.
  all_projects = utils.ResolveEnvVars(utils.ReadYamlFile(args.project_yaml))
  if not all_projects:
    logging.error('Error loading project YAML.')
    return

  logging.info('Validating project YAML against schema.')
  try:
    utils.ValidateConfigYaml(all_projects)
  except jsonschema.exceptions.ValidationError as e:
    logging.error('Error in YAML config: %s', e)
    return

  overall = all_projects['overall']
  audit_logs_project = all_projects.get('audit_logs_project')

  projects = []
  # Always deploy the remote audit logs project first (if present).
  if audit_logs_project:
    projects.append(ProjectConfig(overall=overall,
                                  project=audit_logs_project,
                                  audit_logs_project=None))

  for project_config in all_projects.get('projects', []):
    projects.append(ProjectConfig(overall=overall,
                                  project=project_config,
                                  audit_logs_project=audit_logs_project))

  # If resuming setup from a particular project, skip to that project.
  if args.resume_from_project:
    while (projects and
           projects[0].project['project_id'] != args.resume_from_project):
      skipped = projects.pop(0)
      logging.info('Skipping project %s', skipped.project['project_id'])
    if not projects:
      logging.error('Project not found: %s', args.resume_from_project)

  if projects:
    starting_step = max(1, args.resume_from_step)
    for config in projects:
      logging.info('Setting up project %s', config.project['project_id'])
      if not SetupNewProject(config, starting_step):
        # Don't attempt to deploy additional projects if one project failed.
        return
      starting_step = 1
  else:
    logging.error('No projects to deploy.')


if __name__ == '__main__':
  parser = argparse.ArgumentParser(description='Deploy Projects to GCP.')
  parser.add_argument('--project_yaml', type=str, required=True,
                      help='Location of the project config YAML.')
  parser.add_argument('--resume_from_project', type=str, default='',
                      help=('If the script terminates early, set this to the '
                            'project id that failed to resume from this '
                            'project. Set resume_from_step as well.'))
  parser.add_argument('--resume_from_step', type=int, default=1,
                      help=('If the script terminates early, set this to the '
                            'step that failed to resume from this step.'))
  parser.add_argument('--nodry_run', action='store_false', dest='dry_run',
                      help=argparse.SUPPRESS)
  parser.add_argument('--dry_run', action='store_true',
                      help=('By default, no gcloud commands will be executed. '
                            'Use --nodry_run to execute commands.'))
  parser.set_defaults(dry_run=True)
  parser.add_argument('--gcloud_bin', type=str, default='gcloud',
                      help='Location of the gcloud binary. (default: gcloud)')
  parser.add_argument('--log', type=str, default='INFO',
                      help='The logging level to use. (default: INFO)',
                      choices=set(
                          ['DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL']))

  main(parser.parse_args())
