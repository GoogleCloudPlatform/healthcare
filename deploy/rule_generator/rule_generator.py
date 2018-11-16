"""Utility to generate Forseti scanner rules given project configurations.

Usage:
  rule_generator
      --project_configs="${PROJECT_CONFIGS}"
      --forseti_rules_dir="${FORSETI_RULES_DIR}"
      --forseti_gcp_reader="${FORSETI_GCP_READER}"
      --alsologtostderr
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
from absl import app
from absl import flags
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

FLAGS = flags.FLAGS

flags.DEFINE_string(
    'project_config', None, 'YAML file containing all project configurations.')
flags.DEFINE_string('forseti_rules_dir', None, 'Path of output rules files.')

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


def read_yaml_config(path):
  """Reads a YAML file and return a dictionary of its contents."""
  with open(path) as input_file:
    data = input_file.read()
  try:
    return yaml.load(data)
  except yaml.YAMLError, e:
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

  overall = config_dict['overall']
  # audit_logs_project is omitted if projects use local audit logs.
  audit_logs_project = config_dict.get('audit_logs_project')

  project_configs = []
  if audit_logs_project:
    project_configs.append(
        ProjectConfig(
            overall=overall,
            project=audit_logs_project,
            audit_logs_project=None))
  for project in config_dict.get('projects', []):
    project_configs.append(
        ProjectConfig(
            overall=overall,
            project=project,
            audit_logs_project=audit_logs_project))
  return project_configs, overall


def main(argv):
  if len(argv) > 1:
    raise app.UsageError('Too many command-line arguments.')
  # Load all projects
  project_configs, global_config = load_all_project_configs(
      FLAGS.project_config)

  # Generate rules for each scanner.
  for generator in SCANNER_RULE_GENERATORS:
    rules = generator.generate_rules(project_configs, global_config)
    write_yaml_config(rules, FLAGS.forseti_rules_dir,
                      generator.config_file_name())

if __name__ == '__main__':
  flags.mark_flag_as_required('project_config')
  flags.mark_flag_as_required('forseti_rules_dir')
  app.run(main)
