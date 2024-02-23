#    Copyright 2023 Google LLC

#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at

#        http://www.apache.org/licenses/LICENSE-2.0

#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.


import re
import logging
import sys


from utils import *
from constants import *

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
FHIR_URL = Util.get_hde_fhir_url(HDE_PREFIX, HDE_ENV, FHIR_STORE_LOC)


class Commons:
    """Contains common functions required accross the classes"""

    def __init__(self):
        self.AUTH_QUERY = AuthenticateAndQuery()
        self.RESULTS = None
        self.file_info_df = None
        self.mapping_resources_info_df = None
        self.get_mapping_device_info_df = None
        self.recon_results_df = None
        self.recon_pipelines_df = None

    def get_reconcilated_resources(self, resource_ref_lst):
        """Fetches how given FHIR resource reference ID reconciled into and by which pipeline"""
        RECON_RECORDS = []

        for resource_ref in resource_ref_lst:
            provenance_recon_url = Util.get_reconcilated_resources_url(
                FHIR_URL, resource_ref)
            resources = self.AUTH_QUERY.executeQuery(
                next_page_url=provenance_recon_url)
            recon_results = {}
            recon_results['mapped_ifs_resource_url'] = resource_ref
            if resources['total'] != 0:
                recon_results['recon_into_resource'] = resources['entry'][0]['resource']['target'][0][
                    'reference'] if resources['entry'][0]['resource']['target'][0]['reference'] else "NA"
                recon_results['reconciled_by'] = resources['entry'][0]['resource']['agent'][0]['who'][
                    'reference'] if resources['entry'][0]['resource']['agent'][0]['who']['reference'] else "NA"
            RECON_RECORDS.append(recon_results)
        self.recon_results_df = pd.json_normalize(RECON_RECORDS)

    def populate_file_info(self):
        """Populates result with User provided information."""

        docref_url = Util.get_docref_resource_path_url(FHIR_URL, GCS_FILE_PATH)
        resources = self.AUTH_QUERY.executeQuery(next_page_url=docref_url)
        total_matched_resources = resources["total"]
        file_info = {}
        file_info['src_file_path'] = resources["entry"][0]['resource']['content'][0]['attachment']['url']
        # file_info['src_file_docref_url'] = resources["entry"][0]['fullUrl']
        file_info['src_file_docref_id'] = resources["entry"][0]['resource']['id']
        file_info['src_file_added_ts'] = resources["entry"][0]['resource']['meta']['lastUpdated']
        file_info['src_file_resource_versionId'] = resources["entry"][0]['resource']['meta']['versionId']

        # self.results.append(matched_resource_info)
        if GCS_FILE_PATH.strip().lower().endswith(".csv"):

            # Add details of csv file ingested into BQ table
            csv_file_docref_id = f"DocumentReference/{file_info['src_file_docref_id']}"

            docref_url_for_bq_tbl = Util.get_bq_table_docref_info(
                FHIR_URL, csv_file_docref_id)

            resources = self.AUTH_QUERY.executeQuery(
                next_page_url=docref_url_for_bq_tbl)

            file_info['bq_tbl_docref_id'] = resources['entry'][0]['resource']['target'][0]['reference']
            file_info['ingested_by_device_ref'] = resources['entry'][0]['resource']['agent'][0]['who']['reference']

            docref_url_for_bq = Util.get_docref_resources_url(
                FHIR_URL, str(file_info['bq_tbl_docref_id']).split("/")[-1])
            resources = self.AUTH_QUERY.executeQuery(
                next_page_url=docref_url_for_bq)

            file_info['bq_tbl_name'] = resources['entry'][0]['resource']['content'][0]['attachment']['url']
            file_info['bq_tbl_id'] = resources["entry"][0]['resource']['id']

        df = pd.json_normalize(file_info)
        self.file_info_df = df

    def get_pipeline_info(self, device_ref_lst):
        """ Fetches Pipeline information using Device ID."""
        DEVICES = []

        for device_ref in device_ref_lst:
            device_info = {}
            if device_ref == "NA":
                device_info['device_id'] = "NA"
                device_info['mapping_pipeline_name'] = "NA"
                device_info['mapping_pipeline_start_ts'] = "NA"
            else:
                device_url = Util.get_device_id_url(
                    FHIR_URL, str(device_ref).split("/")[-1])
                resources = self.AUTH_QUERY.executeQuery(
                    next_page_url=device_url)
                device_info['device_id'] = device_ref
                if resources["total"] > 0:
                    for prop in resources["entry"][0]["resource"]["property"]:
                        if "JOB_NAME=" in str(prop["type"]["text"]).strip():
                            device_info['mapping_pipeline_name'] = str(
                                prop["type"]["text"]).split("=")[-1]

                        elif re.match('PIPELINE_START_TIME=(.*)Z', str(prop["type"]["text"]).strip()):
                            device_info['mapping_pipeline_start_ts'] = str(
                                prop["type"]["text"]).split("=")[-1]

                        if "JOB_NAME" in device_info and "PIPELINE_START_TIME=" in device_info:
                            break
            DEVICES.append(device_info)
        df = pd.json_normalize(DEVICES)
        return df

    def get_mapping_df(self, resources):
        mapping_resources_info = []
        if resources['total'] == 0:
            logging.info("No resources have been mapped at this time. Please wait until the provenance-processor adds lineage information to the OFS.")
            sys.exit(0)
        for resource in resources['entry']:
            temp_results = {}
            # temp_results["mapping_provenance_url"] = resource["fullUrl"]
            temp_results["mapping_pipeline_device_ref"] = resource['resource']['agent'][0]['who']['reference']
            # temp_results["mapping_from_source_entity"] = resource["resource"]["entity"][0]["what"]["reference"]
            # temp_results["mapping_provenance_id"] = resource["resource"]["id"]
            temp_results["mapping_ts"] = resource["resource"]["meta"]["lastUpdated"]
            temp_results["mapped_ifs_resource_version"] = resource["resource"]["meta"]["versionId"]
            temp_results["mapped_ifs_resource_url"] = []
            for target_ref in resource["resource"]["target"]:
                temp_results["mapped_ifs_resource_url"].append(
                    target_ref['reference'])
            mapping_resources_info.append(temp_results)
        df = pd.json_normalize(mapping_resources_info).explode(
            'mapped_ifs_resource_url')
        return df

    def get_csv_bq_ingestion_df(self):
        # Add details of csv file ingested into BQ table
        bq_table_info = {}
        csv_file_docref_id = f"DocumentReference/{resources['entry'][i]['resource']['id']}"
        docref_url_for_csv = Util.get_bq_table_docref_info(
            self.FHIR_URL, csv_file_docref_id)
        resources = self.AUTH_QUERY.executeQuery(
            next_page_url=docref_url_for_csv)
        table_info = {}
        table_info['bq_tbl_docref_id'] = resources['entry'][0]['resource']['target'][0]['reference']
        table_info['ingested_by'] = resources['entry'][0]['resource']['agent'][0]['who']
        docref_url_for_bq = Util.get_docref_resources_url(
            self.FHIR_URL, str(table_info['bq_tbl_docref_id']).split("/")[-1])
        resources = self.AUTH_QUERY.executeQuery(
            next_page_url=docref_url_for_bq)
        table_info['bq_tbl_name'] = resources['entry'][0]['resource']['content'][0]['attachment']['url']
        table_info['id'] = resources["entry"][0]['resource']['id']
        csv_file_info['ingested_tbl_info'] = table_info

    def process_all_info(self):
        """Fetches the specific source records/data and respective pipeline of mapped into IFS FHIR resources."""

        for index, row in self.file_info_df.iterrows():
            if GCS_FILE_PATH.strip().lower().endswith(".csv"):
                # Add details of csv file ingested into BQ table
                provenance_url = Util.get_provenance_resource_path_url(
                    FHIR_URL, row['bq_tbl_id'])
            else:
                provenance_url = Util.get_provenance_resource_path_url(
                    FHIR_URL, row['src_file_docref_id'])

            # Fetches provenance resources using DocRef ID and produced harmonized resources Dataframe
            resources = self.AUTH_QUERY.executeQuery(
                next_page_url=provenance_url)
            self.mapping_resources_info_df = self.get_mapping_df(
                resources)

            # Joins harmonized resources Dataframe with source file information
            self.mapping_resources_info_df['tmp'] = 1
            self.file_info_df['tmp'] = 1
            self.RESULTS = pd.merge(
                self.mapping_resources_info_df, self.file_info_df, on=["tmp"])

            # Formatting final Dataframe: RESULTS
            self.RESULTS = self.RESULTS.drop(['tmp'], axis=1)

            # Fetches harmonization pipeline information and craetes a Dataframe
            self.get_mapping_device_info_df = self.get_pipeline_info(
                self.RESULTS['mapping_pipeline_device_ref'].unique())
            self.get_mapping_device_info_df = self.get_mapping_device_info_df.rename(
                columns={'device_id': 'mapping_device_id'})

            # Joins harmonization pipeline Dataframe with final Dataframe: RESULTS
            self.RESULTS = pd.merge(self.RESULTS, self.get_mapping_device_info_df,
                                    left_on='mapping_pipeline_device_ref', right_on='mapping_device_id')

            if RECON_INFO:
                logging.info(f"processing reconciliation information for file: {GCS_FILE_PATH} ...")
                # Fetches respective reconciled final-fhir resource information and craetes a Dataframe
                self.get_reconcilated_resources(
                    self.RESULTS['mapped_ifs_resource_url'].unique())

                # Joins reconciliation resources Dataframe with final Dataframe: RESULTS
                self.RESULTS = pd.merge(self.RESULTS, self.recon_results_df,
                                        left_on='mapped_ifs_resource_url', right_on='mapped_ifs_resource_url')

                # Fetches reconciliation pipeline information and craetes a Dataframe
                self.get_recon_device_info_df = self.get_pipeline_info(
                    self.RESULTS['reconciled_by'].unique())
                self.get_recon_device_info_df = self.get_recon_device_info_df.rename(columns={'device_id': 'recon_device_id', 'mapping_pipeline_name': 'recon_pipeline_name',
                                                                                              'mapping_pipeline_start_ts': 'recon_pipeline_start_ts'})

                # Joins reconciliation pipeline Dataframe with final Dataframe: RESULTS
                self.RESULTS = pd.merge(self.RESULTS, self.get_recon_device_info_df,
                                        left_on='reconciled_by', right_on='recon_device_id')
                
                # Formatting final Dataframe: RESULTS
                self.RESULTS = self.RESULTS.drop(['recon_device_id'], axis=1)

            

            # Write output Dataframe: RESULTS to Bigquery and Local file
            Util.handle_results(self.RESULTS)

            mapping_unique_pipelines = len(self.RESULTS['mapping_pipeline_device_ref'].unique())
            aprox_records_mapped_by_single_pipeline = int(len(self.RESULTS)/mapping_unique_pipelines)
            logging.info(f"Here's the TL:DR for this file: {GCS_FILE_PATH} ...")
            logging.info(f"-------------------------------------------------------------------------------------------------------------------------------------------")
            logging.info(f"This file `{GCS_FILE_PATH}` processed by {mapping_unique_pipelines} pipeline(s). ")
            logging.info(f"The total number of resources mapped by {mapping_unique_pipelines} pipeline(s) from this file into the data store is: {len(self.RESULTS)}.")
            logging.info(f"Each pipeline maps this file `{GCS_FILE_PATH}` to {aprox_records_mapped_by_single_pipeline} resource(s).")
            if OUTPUT_BQ_TBL and OUTPUT_FILE_PATH:
                logging.info(f"Please find the detailed information about the mapped and reconciled resource(s) in BQ table: {OUTPUT_BQ_TBL} and local file: {OUTPUT_FILE_PATH}")
            elif OUTPUT_BQ_TBL and not OUTPUT_FILE_PATH:
                logging.info(f"Please find the detailed information about the mapped and reconciled resource(s) in BQ table: {OUTPUT_BQ_TBL}")
            elif not OUTPUT_BQ_TBL and OUTPUT_FILE_PATH:
                logging.info(f"Please find the detailed information about the mapped and reconciled resource(s) in local file path: {OUTPUT_FILE_PATH}")
            else:
                logging.info(f"Please configure the OUTPUT_BQ_TBL and OUTPUT_FILE_PATH parameters from the file `constants.py` to obtain detailed information.")
            logging.info(f"-------------------------------------------------------------------------------------------------------------------------------------------")


def main():
    commons = Commons()
    logging.info(f"processing OFS resources ...")
    commons.populate_file_info()
    commons.process_all_info()


if __name__ == '__main__':
    main()

