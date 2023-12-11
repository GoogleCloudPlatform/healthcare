import os
import json
import google.auth
from google.auth.transport import requests
# Imports a module to allow authentication using a service account
from google.oauth2 import service_account

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

    def __init__(self):
        self.paginated_results = []

    def getCreds(self):
        credentials = None
        try:
            # Gets credentials from the environment.
            credentials = service_account.Credentials.from_service_account_file(os.environ["GOOGLE_APPLICATION_CREDENTIALS"])
        except KeyError as e:
            credentials, project = google.auth.default()
        return credentials
    
    def getSession(self):
        credentials = self.getCreds()
        scoped_credentials = credentials.with_scopes( ["https://www.googleapis.com/auth/cloud-platform"] )
        # Creates a requests Session object with the credentials.
        session = requests.AuthorizedSession(scoped_credentials)
        return session
    
    def getResults(self, url):
        session = self.getSession()
        return session.get(url)

    
    def executeQuery(self, next_page_url="", method="GET"):
        # session = self.getSession()
        # if method == "GET":
        #     print(f"Get resourceInfo for {resource_path}")
        #     response = session.get(resource_path)
        #     if response.status_code == 200:
        #         response_json = response.json()
        #         if response_json['total'] == len(response_json['entry']):
        #             return response_json
        #         else:
        #             print("Needs pagination to fetch all results")
        #             self
        #             while(response_json['link'][1]['relation']=='next'):
        #                 next_url = response_json['link'][1]['url']
        #                 self.executeQuery(next_url)
        results = {}
        current_page = 1
        while True:
            print(f"retrieving results for page: {current_page} using {next_page_url}")
            response = self.getResults(next_page_url)
            response_json = response.json()

            if not results:
                results.update(response_json)
            else:
                results['entry'].extend(response_json['entry'])
            if response_json['link'][1].get("relation")=='next':
                next_page_url = response_json['link'][1].get("url")
            else:
                break
            current_page += 1
        return results
                    

    
class Util:
    
    @staticmethod
    def get_hde_fhir_url(prefix, env, loc, dataset="healthcare-dataset", fhir_store="operational-fhir-store"):
        base_url = "https://healthcare.googleapis.com/v1"
        return f"{base_url}/projects/{prefix}-{env}-data/locations/{loc}/datasets/{dataset}/fhirStores/{fhir_store}/fhir"
    
    @staticmethod
    def get_docref_resource_path_url(fhir_url, gcs_file_path):
        return f"{fhir_url}/DocumentReference?_content:contains={gcs_file_path}"
    
    @staticmethod
    def get_provenance_resource_path_url(fhir_url, docref_id):
        return f"{fhir_url}/Provenance?entity={docref_id}&_count=1000"

    @staticmethod
    def write_json_to_local_file(json_data):
        with open('data.json', 'w', encoding='utf-8') as f:
            json.dump(json_data, f, ensure_ascii=False, indent=4)

    @staticmethod
    def get_bq_table_docref_info(fhir_url, csv_file_docref_id):
        return f"{fhir_url}/Provenance?_content:contains=ingestion&entity={csv_file_docref_id}"
        
    @staticmethod
    def get_docref_resources_url(fhir_url, docref_id):
        return f"{fhir_url}/DocumentReference?_id={docref_id}"

    @staticmethod
    def get_reconcilated_resources_url(fhir_url, ifs_resource_ref):
        return f"{fhir_url}/Provenance?_content:contains=reconciliation&entity={ifs_resource_ref}"
        



# curl -X GET      
# -H "Authorization: Bearer $(gcloud auth application-default print-access-token)"      
# "https://healthcare.googleapis.com/v1/projects/hde140-uat-12-synth-data/locations/us/datasets/healthcare-dataset/fhirStores/operational-fhir-store/fhir/DocumentReference?_content:contains=gs://hde-accelerators-test-fhir-data" -i