import argparse
import pandas as pd
from utils import *
from constants import *





class CsvFhirHarmonizationTrack:

    def __init__(self, file_format, gcs_file_path, hde_prefix, hde_env, hde_fhir_store_location):
        self.FILE_FORMAT = FILE_FORMAT
        self.GCS_FILE_PATH = GCS_FILE_PATH
        self.HDE_PREFIX = HDE_PREFIX
        self.HDE_ENV = HDE_ENV
        self.FHIR_STORE_LOC = FHIR_STORE_LOC
        self.FHIR_URL = None
        self.results=[]
        self.AUTH_QUERY = AuthenticateAndQuery()

    def get_docref_ids_for_provenance_ref(self):
        self.FHIR_URL = Util.get_hde_fhir_url(self.HDE_PREFIX, self.HDE_ENV, self.FHIR_STORE_LOC)
        docref_url = Util.get_docref_resource_path_url(self.FHIR_URL, self.GCS_FILE_PATH)
        resources = self.AUTH_QUERY.executeQuery(next_page_url=docref_url)
        total_matched_resources = resources["total"]
        for i in range(total_matched_resources):
            csv_file_info = {}
            csv_file_info['file'] = resources["entry"][i]['resource']['content'][0]['attachment']['url']
            csv_file_info['resource_full_url'] = resources["entry"][i]['fullUrl']
            csv_file_info['id'] = resources["entry"][i]['resource']['id']
            csv_file_info['meta'] = resources["entry"][i]['resource']['meta']

            #Add details of csv file ingested into BQ table
            bq_table_info = {}
            csv_file_docref_id = f"DocumentReference/{resources['entry'][i]['resource']['id']}"
            docref_url_for_csv = Util.get_bq_table_docref_info(self.FHIR_URL, csv_file_docref_id)
            resources = self.AUTH_QUERY.executeQuery(next_page_url=docref_url_for_csv)
            table_info = {}
            table_info['bq_tbl_docref_id'] = resources['entry'][0]['resource']['target'][0]['reference']
            table_info['ingested_by'] = resources['entry'][0]['resource']['agent'][0]['who']
            docref_url_for_bq = Util.get_docref_resources_url(self.FHIR_URL, str(table_info['bq_tbl_docref_id']).split("/")[-1])
            resources = self.AUTH_QUERY.executeQuery(next_page_url=docref_url_for_bq)
            table_info['tbl_name'] = resources['entry'][0]['resource']['content'][0]['attachment']['url']
            table_info['id'] = resources["entry"][0]['resource']['id']
            csv_file_info['ingested_tbl_info'] = table_info

            self.results.append(csv_file_info)

    def get_reconcilated_resources(self, resource_ref):
        provenance_recon_url = Util.get_reconcilated_resources_url(self.FHIR_URL, resource_ref)
        resources = self.AUTH_QUERY.executeQuery(next_page_url=provenance_recon_url)
        recon_results={}
        recon_results['reconciled_resource'] = resources['entry'][0]['resource']['target']
        recon_results['reconciled_by'] = resources['entry'][0]['resource']['agent'][0]['who']
        return recon_results


    def get_transformed_fhir_fhir_resources(self):
        for index, source_file_info in enumerate(self.results):
            ifs_resources = {}
            provenance_url = Util.get_provenance_resource_path_url(self.FHIR_URL, source_file_info['ingested_tbl_info']['id'])
            print(provenance_url)
            resources = self.AUTH_QUERY.executeQuery(next_page_url=provenance_url)
            total_transformed_ifs_resources = resources["total"]
            total_resources_entry = len(resources['entry'])

            transformed_ifs_resources = {}
            self.results[index]['ingested_tbl_info']['total_resoruces_transformed_ifs'] = resources["total"]
            self.results[index]['ingested_tbl_info']['total_resources_entry'] = total_resources_entry
            self.results[index]['ingested_tbl_info']['resources_transformed_ifs'] = []
            print(f"iterating loop over {total_transformed_ifs_resources} times")

            for i in range(total_resources_entry):
                ifs_resource_info = {}
                ifs_resource_info['transormed_resources'] = []
                for ifs_resource in resources['entry'][i]['resource']['target']:
                     temp_ifs_resource_info = {}
                     temp_ifs_resource_info['reference'] = ifs_resource['reference']
                     
                     #fetch reconcilation info for this resource
                     temp_ifs_resource_info['reconciled_into'] = self.get_reconcilated_resources(ifs_resource['reference'])
                     ifs_resource_info['transormed_resources'].append(temp_ifs_resource_info)
                ifs_resource_info['transformed_by'] = resources['entry'][i]['resource']['agent'][0]['who']
                self.results[index]['ingested_tbl_info']['resources_transformed_ifs'].append(ifs_resource_info)

    def process(self):
        print("processing audit logs for fhir to fhir harmonization")
        self.get_docref_ids_for_provenance_ref()
        # self.get_transformed_fhir_fhir_resources()
        # elif str(self.FILE_FORMAT).strip().lower == "csv":
        #     self.get_docref_ids()
        #     self.get_transformed_csv_fhir_resources()
        Util.write_json_to_local_file(self.results)


class FhirFhirHarmonizationTrack:
    def __init__(self, **kwargs):
        self.FILE_FORMAT = FILE_FORMAT
        self.GCS_FILE_PATH = GCS_FILE_PATH
        self.HDE_PREFIX = HDE_PREFIX
        self.HDE_ENV = HDE_ENV
        self.FHIR_STORE_LOC = FHIR_STORE_LOC
        self.FHIR_URL = None
        self.results=[]
        self.AUTH_QUERY = AuthenticateAndQuery()

    def get_docref_ids_for_provenance_ref(self):
        '''fetching DocumentReference ID for reference while querying Provenance resource'''
        self.FHIR_URL = Util.get_hde_fhir_url(self.HDE_PREFIX, self.HDE_ENV, self.FHIR_STORE_LOC)
        docref_url = Util.get_docref_resource_path_url(self.FHIR_URL, self.GCS_FILE_PATH)
        resources = self.AUTH_QUERY.executeQuery(next_page_url=docref_url)
        total_matched_resources = resources["total"]
        for i in range(total_matched_resources):
            matched_resource_info = {}
            matched_resource_info['file'] = resources["entry"][i]['resource']['content'][0]['attachment']['url']
            matched_resource_info['resource_full_url'] = resources["entry"][i]['fullUrl']
            matched_resource_info['id'] = resources["entry"][i]['resource']['id']
            matched_resource_info['meta'] = resources["entry"][i]['resource']['meta']
            self.results.append(matched_resource_info)

    def get_reconcilated_resources(self, resource_ref):
        provenance_recon_url = Util.get_reconcilated_resources_url(self.FHIR_URL, resource_ref)
        resources = self.AUTH_QUERY.executeQuery(next_page_url=provenance_recon_url)
        recon_results={}
        recon_results['reconciled_resource'] = resources['entry'][0]['resource']['target']
        recon_results['reconciled_by'] = resources['entry'][0]['resource']['agent'][0]['who']
        return recon_results


    def get_transformed_fhir_fhir_resources(self):
        for index, source_file_info in enumerate(self.results):
            ifs_resources = {}
            provenance_url = Util.get_provenance_resource_path_url(self.FHIR_URL, source_file_info['id'])
            print(provenance_url)
            resources = self.AUTH_QUERY.executeQuery(next_page_url=provenance_url)
            total_transformed_ifs_resources = resources["total"]
            total_resources_entry = len(resources['entry'])

            transformed_ifs_resources = {}
            self.results[index]['total_resoruces_transformed_ifs'] = resources["total"]
            self.results[index]['total_resources_entry'] = total_resources_entry
            self.results[index]['resources_transformed_ifs'] = []
            print(f"iterating loop over {total_transformed_ifs_resources} times")

            for i in range(total_resources_entry):
                ifs_resource_info = {}
                ifs_resource_info['transormed_resources'] = []
                for ifs_resource in resources['entry'][i]['resource']['target']:
                     temp_ifs_resource_info = {}
                     temp_ifs_resource_info['reference'] = ifs_resource['reference']
                     #fetch reconcilation info for this resource
                     temp_ifs_resource_info['reconciled_into'] = self.get_reconcilated_resources(ifs_resource['reference'])
                     ifs_resource_info['transormed_resources'].append(temp_ifs_resource_info)
                ifs_resource_info['transformed_by'] = resources['entry'][i]['resource']['agent'][0]['who']
                self.results[index]['resources_transformed_ifs'].append(ifs_resource_info)

    def process(self):
        print("processing audit logs for fhir to fhir harmonization")
        self.get_docref_ids_for_file()
        self.get_fhir_fhir_transformed_resources()
        Util.write_json_to_local_file(self.results)

def main():
    if str(FILE_FORMAT).strip().lower() == "ndjson":
        processing_obj = fhir_fhir_harmonization_track(args)
        processing_obj.process()
    if str(FILE_FORMAT).strip().lower() == "csv":
        processing_obj = csv_fhir_harmonization_track(file_format=args.file_format, 
        gcs_file_path=args.gcs_file_path, hde_prefix=args.hde_prefix, hde_env=args.hde_env, hde_fhir_store_location=args.hde_fhir_store_location)
        processing_obj.process()
    if str(FILE_FORMAT).strip().lower() == "xml":
        processing_obj = cda_fhir_harmonization_track(args)
        processing_obj.process()


if __name__ == '__main__':
    main()    
