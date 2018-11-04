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
import copy
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


def create_new_project(config):
  """Creates the new GCP project."""
  logging.info('Creating a new GCP project...')
  project_id = config.project['project_id']
  org_id = config.overall.get('organization_id')
  folder_id = config.overall.get('folder_id')

  create_project_command = ['projects', 'create', project_id]
  if folder_id:
    create_project_command.extend(['--folder', folder_id])
  elif org_id:
    create_project_command.extend(['--organization', org_id])
  else:
    logging.info('Deploying without a parent organization or folder.')
  # Create the new project.
  utils.run_gcloud_command(create_project_command, project_id=None)


def setup_billing(config):
  """Sets the billing account for this project."""
  logging.info('Setting up billing...')
  billing_acct = config.overall['billing_account']
  project_id = config.project['project_id']
  # Set the appropriate billing account for this project:
  utils.run_gcloud_command(['beta', 'billing', 'projects', 'link', project_id,
                            '--billing-account', billing_acct],
                           project_id=None)


def enable_deployment_manager(config):
  """Enables Deployment manager, with role/owners for its service account."""
  logging.info('Setting up Deployment Manager...')
  project_id = config.project['project_id']

  # Enabled Deployment Manger and Cloud Resource Manager for this project.
  utils.run_gcloud_command(['services', 'enable', 'deploymentmanager',
                            'cloudresourcemanager.googleapis.com'],
                           project_id)

  # Grant deployment manager service account (temporary) owners access.
  dm_service_account = utils.get_deployment_manager_service_account(project_id)
  utils.run_gcloud_command(['projects', 'add-iam-policy-binding', project_id,
                            '--member', dm_service_account,
                            '--role', 'roles/owner'],
                           project_id=None)


def deploy_gcs_audit_logs(config):
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
  utils.create_new_deployment(dm_template_dict, deployment_name,
                              audit_project_id)


def deploy_project_resources(config):
  """Deploys resources into the new data project."""
  logging.info('Deploying Project resources...')
  setup_account = utils.get_gcloud_user()
  has_organization = bool(config.overall.get('organization_id'))
  project_id = config.project['project_id']
  dm_service_account = utils.get_deployment_manager_service_account(project_id)

  # Build a deployment config for the data_project.py deployment manager
  # template.
  properties = copy.deepcopy(config.project)
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
  utils.create_new_deployment(dm_template_dict, 'data-project-deployment',
                              project_id)

  # Remove Owners role from the DM service account.
  utils.run_gcloud_command(['projects', 'remove-iam-policy-binding', project_id,
                            '--member', dm_service_account,
                            '--role', 'roles/owner'],
                           project_id=None)


def deploy_bigquery_audit_logs(config):
  """Deploys the BigQuery audit logs dataset, if used."""
  data_project_id = config.project['project_id']
  logs_dataset = copy.deepcopy(
      config.project['audit_logs']['logs_bigquery_dataset'])
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
  logs_dataset['log_sink_service_account'] = utils.get_log_sink_service_account(
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
  utils.create_new_deployment(dm_template_dict, deployment_name,
                              audit_project_id)


def create_compute_images(config):
  """Creates new Compute Engine VM images if specified in config."""
  gce_instances = config.project.get('gce_instances')
  if not gce_instances:
    logging.info('No GCS Images required.')
    return
  project_id = config.project['project_id']

  for instance in gce_instances:
    custom_image = instance.get('custom_boot_image')
    if not custom_image:
      logging.info('Using existing compute image %s.',
                   instance['existing_boot_image'])
      continue
    # Check if custom image already exists.
    if utils.run_gcloud_command(
        ['compute', 'images', 'list', '--no-standard-images',
         '--filter', 'name={}'.format(custom_image['image_name']),
         '--format', 'value(name)'],
        project_id=project_id):
      logging.info('Image %s already exists, skipping image creation.',
                   custom_image['image_name'])
      continue
    logging.info('Creating VM Image %s.', custom_image['image_name'])

    # Create VM image using gcloud rather than deployment manager so that the
    # deployment manager service account doesn't need to be granted access to
    # the image GCS bucket.
    image_uri = 'gs://' + custom_image['gcs_path']
    utils.run_gcloud_command(
        ['compute', 'images', 'create', custom_image['image_name'],
         '--source-uri', image_uri],
        project_id=project_id)


def create_compute_vms(config):
  """Creates new GCE VMs and firewall rules if specified in config."""
  if 'gce_instances' not in config.project:
    logging.info('No GCS VMs required.')
    return
  project_id = config.project['project_id']
  logging.info('Creating GCS VMs.')

  # Enable OS Login for VM SSH access.
  utils.run_gcloud_command(['compute', 'project-info', 'add-metadata',
                            '--metadata', 'enable-oslogin=TRUE'],
                           project_id=project_id)

  gce_instances = []
  for instance in config.project['gce_instances']:
    if 'existing_boot_image' in instance:
      image_name = instance['existing_boot_image']
    else:
      image_name = (
          'global/images/' + instance['custom_boot_image']['image_name'])

    gce_template_dict = {
        'name': instance['name'],
        'zone': instance['zone'],
        'machine_type': instance['machine_type'],
        'boot_image_name': image_name,
        'start_vm': instance['start_vm']
    }
    startup_script_str = instance.get('startup_script')
    if startup_script_str:
      gce_template_dict['metadata'] = {
          'items': [{
              'key': 'startup-script',
              'value': startup_script_str
          }]
      }
    gce_instances.append(gce_template_dict)

  deployment_name = 'gce-vms'
  dm_template_dict = {
      'imports': [{'path': 'gce_vms.py'}],
      'resources': [{
          'type': 'gce_vms.py',
          'name': deployment_name,
          'properties': {
              'gce_instances': gce_instances,
              'firewall_rules': config.project.get('gce_firewall_rules', []),
          }
      }]
  }
  utils.create_new_deployment(dm_template_dict, deployment_name, project_id)


def create_stackdriver_account(config):
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
    if not utils.wait_for_yes_no('Account created [y/N]?'):
      logging.warning('Skipping creation of Stackdriver Account.')
      return

    # Verify account was created.
    try:
      utils.run_gcloud_command(['alpha', 'monitoring', 'policies', 'list'],
                               project_id)
      return
    except utils.GcloudRuntimeError as e:
      logging.error('Error reading Stackdriver account %s', e)
      print('Could not find Stackdriver account.')


def create_alerts(config):
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
  channel = utils.create_notification_channel(alert_email, project_id)

  logging.info('Creating Stackdriver alerts.')
  utils.create_alert_policy(
      'global', 'iam-policy-change-count', 'IAM Policy Change Alert',
      ('This policy ensures the designated user/group is notified when IAM '
       'policies are altered.'), channel, project_id)

  utils.create_alert_policy(
      'gcs_bucket', 'bucket-permission-change-count',
      'Bucket Permission Change Alert',
      ('This policy ensures the designated user/group is notified when '
       'bucket/object permissions are altered.'), channel, project_id)

  for data_bucket in config.project.get('data_buckets', []):
    # Every bucket with 'expected_users' has an expected-access alert.
    if 'expected_users' in data_bucket:
      bucket_name = project_id + data_bucket['name_suffix']
      metric_name = 'unexpected-access-' + bucket_name
      utils.create_alert_policy(
          'gcs_bucket', metric_name,
          'Unexpected Access to {} Alert'.format(bucket_name),
          ('This policy ensures the designated user/group is notified when '
           'bucket {} is accessed by an unexpected user.'.format(bucket_name)),
          channel, project_id)


def create_dataproc(config):
  """Creates new dataproc cluster config."""
  if 'dataproc' not in config.project:
    logging.info('No dataproc cluster required.')
    return
  project_id = config.project['project_id']
  logging.info('Creating dataproc cluster')

  dataproc_clusters = []
  for cluster in config.project['dataproc']:
    zone = 'https://www.googleapis.com/compute/v1/projects/{}/zones/{}'.format(project_id, cluster['zone'])
    machine_type = 'https://www.googleapis.com/compute/v1/projects/{}/zones/{}/machineTypes/{}'.format(project_id, cluster['zone'],cluster['machine_type'])
    scripts = []
    for script in cluster['init_scripts']:
        scripts.append(script['name'])
        logging.info('script is {}'.format(script['name'])) 
    tags = []
    for tag in cluster['tags']:
        tags.append(tag['name'])

    cluster_template_dict = {
        'name': cluster['name'],
        'workernum': cluster['workernum'],
        'masternum': cluster['masternum'],
        'region': cluster['region'],
        'zone': zone,
        'tags': tags,
        'machine_type': machine_type,
        'init_scripts': scripts
    }
    dataproc_clusters.append(cluster_template_dict)

  deployment_name = 'create-dataproc'
  dm_template_dict = {
      'imports': [{'path': 'create_dataproc.py'}],
      'resources': [{
          'type': 'create_dataproc.py',
          'name': deployment_name,
          'properties': {
              'dataproc_clusters': dataproc_clusters,
          }
      }]
  }
  utils.create_new_deployment(dm_template_dict, deployment_name, project_id)

_SETUP_STEPS = [
    create_new_project,
    setup_billing,
    enable_deployment_manager,
    deploy_gcs_audit_logs,
    deploy_project_resources,
    deploy_bigquery_audit_logs,
    create_compute_images,
    create_compute_vms,
    create_stackdriver_account,
    create_alerts,
    create_dataproc
]


def setup_new_project(config, starting_step):
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


def add_generated_fields(project):
  """Adds a generated_fields block to a project definition, if not already set.

  Args:
    project (dict): Config dictionary of a single project.
  """
  if 'generated_fields' not in project:
    project['generated_fields'] = {
        'project_number': utils.get_project_number(project['project_id']),
        'log_sink_service_account': utils.get_log_sink_service_account(
            _LOG_SINK_NAME, project['project_id'])
    }


def main(args):
  logging.basicConfig(level=getattr(logging, args.log))
  utils.GCLOUD_OPTIONS = utils.GcloudOptions(
      dry_run=args.dry_run, gcloud_bin=args.gcloud_bin)

  # Output YAML will rearrange fields and remove comments, so do a basic check
  # against accidental overwriting.
  if args.project_yaml == args.output_yaml_path:
    logging.error('output_yaml_path cannot overwrite project_yaml.')
    return

  # Read and parse the project configuration YAML file.
  all_projects = utils.resolve_env_vars(utils.read_yaml_file(args.project_yaml))
  if not all_projects:
    logging.error('Error loading project YAML.')
    return

  logging.info('Validating project YAML against schema.')
  try:
    utils.validate_config_yaml(all_projects)
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
      if not setup_new_project(config, starting_step):
        # Don't attempt to deploy additional projects if one project failed.
        return
      starting_step = 1
  else:
    logging.error('No projects to deploy.')

  # After all projects are deployed, fill in unset generated fields for all
  # projects and save the final YAML file.
  if args.output_yaml_path:
    if audit_logs_project:
      add_generated_fields(audit_logs_project)
    for project in all_projects.get('projects', []):
      add_generated_fields(project)

    utils.write_yaml_file(all_projects, args.output_yaml_path)


def get_parser():
  """Returns an argument parser."""
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
  parser.add_argument('--output_yaml_path', type=str, default='',
                      help=('If provided, save a new YAML file with any '
                            'environment variables substituted and generated '
                            'fields populated. This must be different to '
                            'project_yaml.'))
  parser.add_argument('--gcloud_bin', type=str, default='gcloud',
                      help='Location of the gcloud binary. (default: gcloud)')
  parser.add_argument('--log', type=str, default='INFO',
                      help='The logging level to use. (default: INFO)',
                      choices=set(
                          ['DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL']))
  return parser

if __name__ == '__main__':
  main(get_parser().parse_args())
