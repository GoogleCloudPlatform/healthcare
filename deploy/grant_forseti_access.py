r"""Script to grant the Forseti service account access to a project.

Usage:
  python3 grant_forseti_access.py \
    --project_id=some-project \
    --forseti_service_account=forseti@my-forseti-project.iam.gserviceaccount.com
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import logging

from utils import forseti
from utils import runner


def main(args):
  logging.basicConfig(level=getattr(logging, args.log))
  runner.DRY_RUN = args.dry_run
  runner.GCLOUD_BINARY = args.gcloud_bin
  forseti.grant_access(args.project_id, args.forseti_service_account)

if __name__ == '__main__':
  parser = argparse.ArgumentParser(description='Grant access for Forseti.')
  parser.add_argument('--project_id', type=str, required=True,
                      help='GCP Project ID of the project to add access to.')
  parser.add_argument('--forseti_service_account', type=str, required=True,
                      help='Forseti Service account to grant access to.')
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
