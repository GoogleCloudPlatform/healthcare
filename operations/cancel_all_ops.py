# Copyright 2022 Google LLC.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Cancels all pending operations."""

import argparse
import time

# Imports the Google API Discovery Service. To install, use:
# pip install --upgrade google-api-python-client google-auth-httplib2 \
#   google-auth-oauthlib
import googleapiclient
from googleapiclient import discovery

parser = argparse.ArgumentParser(description="Cancel pending operations.")
parser.add_argument("--project_id", required=True)
parser.add_argument("--location", required=True)
parser.add_argument("--dataset_id", required=True)
parser.add_argument("--dry_run", required=False, help="List the operations " +
                    "that would have been canceled, but don't cancel them.")


def cancel_operations(project_id, location, dataset_id, dry_run):
  """Cancels all operations for the given dataset."""

  api_version = "v1"
  service_name = "healthcare"
  # Returns an authorized API client by discovering the Healthcare API
  # and using GOOGLE_APPLICATION_CREDENTIALS environment variable.
  client = discovery.build(service_name, api_version)

  dataset_parent = "projects/{}/locations/{}".format(project_id, location)
  dataset_name = "{}/datasets/{}".format(dataset_parent, dataset_id)

  if dry_run:
    print("Listing all incomplete ops for dataset {}".format(dataset_name))
  else:
    print("Cancelling all incomplete ops for dataset {}".format(dataset_name))

  op_client = client.projects().locations().datasets().operations()
  page_token = ""
  while True:
    request = op_client.list(name=dataset_name, pageToken=page_token)
    response = request.execute()

    ops = response.get("operations", [])
    for op in ops:
      if op.get("done", False):
        continue
      if dry_run:
        print("Would cancel op {}.".format(op["name"]))
        continue
      try:
        time.sleep(0.2)  # Avoid getting rate-limited.
        op_client.cancel(name=op["name"]).execute()
        print("Cancelled op {}.".format(op["name"]))
      except googleapiclient.errors.HttpError as err:
        details = err.error_details
        if (err.resp.status == 400 and
            len(details) and "detail" in details[0] and
            "operation has already completed" in details[0]["detail"]):
          continue
        raise

    # Check for another page of ListOperations results.
    if "nextPageToken" not in response:
      break
    page_token = response["nextPageToken"]

  return response


def main():
  args = vars(parser.parse_args())
  cancel_operations(args["project_id"], args["location"], args["dataset_id"],
                    args["dry_run"])

if __name__ == "__main__":
  main()
