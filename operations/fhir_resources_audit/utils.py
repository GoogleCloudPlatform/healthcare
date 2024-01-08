
import os
import json
import pandas as pd
import sys
import logging

import google.auth
from google.auth.transport import requests

# Imports a module to allow authentication using a service account
from google.oauth2 import service_account

# Imports configs from constants.py
from constants import *

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')

class HttpMethods:

    def __init__(self, base_url):
        self.base_url = base_url

    def get(self, path, **kwargs):
        """Makes a GET request to the specified path."""
        url = self.base_url + path
        response = requests.get(url, **kwargs)
        return response

    def post(self, path, data, **kwargs):
        """Makes a POST request to the specified path with the specified data."""
        url = self.base_url + path
        response = requests.post(url, data=data, **kwargs)
        return response

    def put(self, path, data, **kwargs):
        """Makes a PUT request to the specified path with the specified data."""
        url = self.base_url + path
        response = requests.put(url, data=data, **kwargs)
        return response

    def delete(self, path, **kwargs):
        """Makes a DELETE request to the specified path."""
        url = self.base_url + path
        response = requests.delete(url, **kwargs)
        return response


class AuthenticateAndQuery:
    """Authenticates credentials from the environment and executes query."""

    def __init__(self):
        self.paginated_results = []

    def getCreds(self):
        """Creates a service object using the service account credentials JSON file."""
        credentials = None
        try:
            credentials = service_account.Credentials.from_service_account_file(
                os.environ["GOOGLE_APPLICATION_CREDENTIALS"])
        except KeyError as e:
            credentials, project = google.auth.default()
        return credentials

    def getSession(self):
        """Creates a requests Session object with the credentials. """
        credentials = self.getCreds()
        scoped_credentials = credentials.with_scopes(
            ["https://www.googleapis.com/auth/cloud-platform"])
        session = requests.AuthorizedSession(scoped_credentials)
        return session

    def getResults(self, url):
        """ Executes GET call on given url using session object"""
        session = self.getSession()
        return session.get(url)

    def executeQuery(self, next_page_url="", method="GET"):
        """Formats results(paginated) from the GET call on given url"""
        try:
            results = {}
            current_page = 1
            while True:
                logging.info(
                    f"retrieving results for page: {current_page} using url:{next_page_url}")
                response = self.getResults(next_page_url)
                response_json = response.json()

                if not results:
                    results.update(response_json)
                else:
                    results['entry'].extend(response_json['entry'])
                if response_json['link'][1].get("relation") == 'next':
                    next_page_url = response_json['link'][1].get("url")
                else:
                    break
                current_page += 1
            return results
        except KeyError as e:
            logging.error("Please re-check properties in constants.py to resolve the issue")
            sys.exit(0)


class Util:

    @staticmethod
    def get_hde_fhir_url(prefix, env, loc, dataset="healthcare-dataset", fhir_store="operational-fhir-store"):
        """Generates fhir store url using prefix, env, location fields"""
        base_url = BASE_CHC_API_URL
        return f"{base_url}/projects/{prefix}-{env}-data/locations/{loc}/datasets/{dataset}/fhirStores/{fhir_store}/fhir"

    @staticmethod
    def get_docref_resource_path_url(fhir_url, gcs_file_path):
        """Generates url to search string in DocumentReference resource"""
        return f"{fhir_url}/DocumentReference?_content:contains={gcs_file_path}"

    @staticmethod
    def get_provenance_resource_path_url(fhir_url, docref_id):
        """Generates url to search resources with particular DocumentReference ID and _count is added for pagination"""
        return f"{fhir_url}/Provenance?entity={docref_id}&_count=1000"

    @staticmethod
    def get_device_id_url(fhir_url, device_id):
        """Generates url to search resources with particular Device ID"""
        return f"{fhir_url}/Device?_id={device_id}"

    @staticmethod
    def write_json_to_local_file(json_data):
        """Writes output to local output file in local directory"""
        output_file_path = OUTPUT_FILE_PATH if OUTPUT_FILE_PATH else "output.json"
        with open(output_file_path, 'w', encoding='utf-8') as f:
            json.dump(json_data, f, ensure_ascii=False, indent=4)

    @staticmethod
    def handle_results(final_df):
        if OUTPUT_BQ_TBL:
            """Writes output to BQ Table"""
            table_id = str(OUTPUT_BQ_TBL).strip().split(":")[1]
            project_id = str(OUTPUT_BQ_TBL).strip().split(":")[0]
            final_df.to_gbq(table_id,
                            project_id=project_id, if_exists='replace')
        if OUTPUT_FILE_PATH:
            """Writes output to local output file in local directory"""
            output_file_path = OUTPUT_FILE_PATH if OUTPUT_FILE_PATH else "output.json"
            # final_df.to_json(output_file_path, orient = 'split', compression = 'infer', index = 'true')
            result = final_df.to_json(orient="records")
            parsed_op_data = json.loads(result)
            with open(output_file_path, 'w', encoding='utf-8') as f:
                json.dump(parsed_op_data, f, ensure_ascii=False, indent=4)

    @staticmethod
    def get_bq_table_docref_info(fhir_url, csv_file_docref_id):
        """Generates url to search DocumentReference ID for the BQ table in CSV harmonization"""
        return f"{fhir_url}/Provenance?_content:contains=ingestion&entity={csv_file_docref_id}"

    @staticmethod
    def get_docref_resources_url(fhir_url, docref_id):
        """Generates url to search DocumentReference resources with specific ID"""
        return f"{fhir_url}/DocumentReference?_id={docref_id}"

    @staticmethod
    def get_reconcilated_resources_url(fhir_url, ifs_resource_ref):
        """Generates url to search reconciled resources using Provenance resource"""
        return f"{fhir_url}/Provenance?_content:contains=reconciliation&entity={ifs_resource_ref}"

