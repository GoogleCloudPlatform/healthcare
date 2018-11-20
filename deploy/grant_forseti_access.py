r"""Script to grant the Forseti service account access to a project.

Usage:
  bazel run :grant_forseti_access -- \
    --project_id=some-project \
    --forseti_service_account=forseti@my-forseti-project.iam.gserviceaccount.com
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl import app
from absl import flags

from deploy.utils import forseti
from deploy.utils import runner

FLAGS = flags.FLAGS

flags.DEFINE_string('project_id', None,
                    'GCP Project ID of the project to add access to.')
flags.DEFINE_string('forseti_service_account', None,
                    'Forseti Service account to grant access to.')
flags.DEFINE_bool('dry_run', True,
                  ('By default, no gcloud commands will be executed. '
                   'Use --nodry_run to execute commands.'))
flags.DEFINE_string('gcloud_bin', 'gcloud',
                    'Location of the gcloud binary. (default: gcloud)')


def main(argv):
  del argv  # Unused.
  runner.DRY_RUN = FLAGS.dry_run
  runner.GCLOUD_BINARY = FLAGS.gcloud_bin
  forseti.grant_access(FLAGS.project_id, FLAGS.forseti_service_account)

if __name__ == '__main__':
  flags.mark_flag_as_required('project_id')
  flags.mark_flag_as_required('forseti_service_account')
  app.run(main)
