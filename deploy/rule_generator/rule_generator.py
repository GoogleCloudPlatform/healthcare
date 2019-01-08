"""Utility to generate Forseti scanner rules given project configurations."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
import posixpath

from absl import logging
from backports import tempfile

from deploy.rule_generator.project_config import ProjectConfig
from deploy.rule_generator.scanners.audit_logging_scanner_rules import AuditLoggingScannerRules
from deploy.rule_generator.scanners.bigquery_scanner_rules import BigQueryScannerRules
from deploy.rule_generator.scanners.bucket_scanner_rules import BucketScannerRules
from deploy.rule_generator.scanners.cloudsql_scanner_rules import CloudSqlScannerRules
from deploy.rule_generator.scanners.enabled_apis_scanner_rules import EnabledApisScannerRules
from deploy.rule_generator.scanners.iam_scanner_rules import IamScannerRules
from deploy.rule_generator.scanners.lien_scanner_rules import LienScannerRules
from deploy.rule_generator.scanners.location_scanner_rules import LocationScannerRules
from deploy.rule_generator.scanners.log_sink_scanner_rules import LogSinkScannerRules
from deploy.rule_generator.scanners.resource_scanner_rules import ResourceScannerRules
from deploy.utils import runner
from deploy.utils import utils


# All Scanner Rule Generators to use.
SCANNER_RULE_GENERATORS = [
    AuditLoggingScannerRules(),
    BigQueryScannerRules(),
    BucketScannerRules(),
    CloudSqlScannerRules(),
    EnabledApisScannerRules(),
    IamScannerRules(),
    LienScannerRules(),
    LocationScannerRules(),
    LogSinkScannerRules(),
    ResourceScannerRules(),
]


def run(deployment_config, output_path=None):
  """Run the rule generator.

  Generate rules for all supported scanners based on the given deployment config
  and write them in the given output directory.

  Args:
    deployment_config(dict): The loaded yaml deployment config.
    output_path (str): Path to a local directory or a GCS bucket
      path starting with gs://.

  Raises:
    ValueError: If no output_path given AND no forseti config in the
      deployment_config.
  """
  if not output_path:
    output_path = deployment_config.get('forseti', {}).get(
        'generated_fields', {}).get('server_bucket')
    if not output_path:
      raise ValueError(
          ('Must provide an output path or set the "forseti_server_bucket" '
           'field in the overall generated_fields'))

  if output_path.startswith('gs://'):
    # output path is a GCS bucket
    with tempfile.TemporaryDirectory() as tmp_dir:
      _write_rules(deployment_config, tmp_dir)
      logging.info('Copying rules files to %s', output_path)
      runner.run_command([
          'gsutil', 'cp',
          os.path.join(tmp_dir, '*.yaml'),
          posixpath.join(output_path, 'rules'),
      ])
  else:
    # output path is a local directory
    _write_rules(deployment_config, output_path)


def _write_rules(deployment_config, directory):
  """Write a rules yaml file for each generator to the given directory."""
  project_configs, global_config = get_all_project_configs(deployment_config)
  for generator in SCANNER_RULE_GENERATORS:
    config_file_name = generator.config_file_name()
    logging.info('Generating rules for %s', config_file_name)
    rules = generator.generate_rules(project_configs, global_config)
    path = os.path.join(directory, config_file_name)
    utils.write_yaml_file(rules, path)


def get_all_project_configs(config_dict):
  """Returns a list of ProjectConfigs and an overall config dictionary."""

  # forseti is omitted if there is no forseti config
  forseti = config_dict.get('forseti')

  # audit_logs_project is omitted if projects use local audit logs.
  audit_logs_project = config_dict.get('audit_logs_project')

  project_configs = []

  if audit_logs_project:
    project_configs.append(
        ProjectConfig(
            project=audit_logs_project,
            audit_logs_project=None,
            forseti=forseti))

  project_dicts = config_dict.get('projects', [])

  forseti_project = config_dict.get('forseti', {}).get('project')
  if forseti_project:
    # insert forseti project before regular projects so that the forseti rules
    # show up first
    project_dicts.insert(0, forseti_project)

  for project in project_dicts:
    project_configs.append(
        ProjectConfig(
            project=project,
            audit_logs_project=audit_logs_project,
            forseti=forseti))
  return project_configs, config_dict['overall']
