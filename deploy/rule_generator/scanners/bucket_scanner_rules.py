"""Rule Generator for Forseti's GCS Bucket Scanner.

Produces global rules blacklisting all ACL rules on GCS buckets. IAM roles for
buckets are handled by the IAM scanner.
"""
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from deploy.rule_generator.scanners import base_scanner_rules


class BucketScannerRules(base_scanner_rules.BaseScannerRules):
  """Scanner rule generator for the GCS Bucket ACL scanner."""

  def config_file_name(self):
    return 'bucket_rules.yaml'

  def _get_global_rules(self, global_config, project_configs):
    del global_config, project_configs  # Unused.
    # The bucket scanner only requires a single, global rule.
    return [{
        'name': 'Disallow all acl rules, only allow IAM.',
        'bucket': '*',
        'entity': '*',
        'email': '*',
        'domain': '*',
        'role': '*',
        'resource': [{
            'resource_ids': ['*'],
        }]
    }]
