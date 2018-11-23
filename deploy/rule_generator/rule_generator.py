"""Utility to generate Forseti scanner rules given project configurations."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
from absl import logging

import yaml

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
]


def run(config_path, output_dir):
  """Run the rule generator.

  Generate rules for all supported scanners based on the given deployment config
  and write them in the given output directory.

  Args:
    config_path (str): Path to the deployment config yaml file.
    output_dir (str): Path to the output directory.
  """
  project_configs, global_config = load_all_project_configs(config_path)

  for generator in SCANNER_RULE_GENERATORS:
    rules = generator.generate_rules(project_configs, global_config)
    write_yaml_config(rules, output_dir, generator.config_file_name())


def read_yaml_config(path):
  """Reads a YAML file and return a dictionary of its contents."""
  with open(path) as input_file:
    data = input_file.read()
  try:
    return yaml.load(data)
  except yaml.YAMLError as e:
    raise ValueError('Error parsing YAML file %s: %s' % (path, e))


def write_yaml_config(config, config_dir, filename):
  """Writes a config dict as yaml to config_dir/filename."""
  if not os.path.isdir(config_dir):
    os.makedirs(config_dir)
  config_filename = os.path.join(config_dir, filename)
  # Don't use aliases in the YAML output.
  yaml.Dumper.ignore_aliases = lambda self, data: True
  with open(config_filename, 'w') as f:
    f.write(yaml.dump(config, default_flow_style=False))
  logging.info('Wrote config to %s', config_filename)


def load_all_project_configs(config_file):
  """Returns a list of ProjectConfigs and an overall config dictionary."""
  config_dict = read_yaml_config(config_file)

  project_configs = []
  overall = config_dict['overall']

  # audit_logs_project is omitted if projects use local audit logs.
  audit_logs_project = config_dict.get('audit_logs_project')
  if audit_logs_project:
    project_configs.append(
        ProjectConfig(
            overall=overall,
            project=audit_logs_project,
            audit_logs_project=None))

  project_dicts = config_dict.get('projects', [])

  forseti_project = config_dict.get('forseti', {}).get('project')
  if forseti_project:
    # insert forseti project before regular projects so that the forseti rules
    # show up first
    project_dicts.insert(0, forseti_project)

  for project in project_dicts:
    project_configs.append(
        ProjectConfig(
            overall=overall,
            project=project,
            audit_logs_project=audit_logs_project))
  return project_configs, overall

