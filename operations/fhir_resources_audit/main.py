import re

from utils import *
from constants import *


FHIR_URL = Util.get_hde_fhir_url(HDE_PREFIX, HDE_ENV, FHIR_STORE_LOC)



class Commons:
    """Contains common functions required accross the classes"""

    def __init__(self):
        self.AUTH_QUERY = AuthenticateAndQuery()
        self.RESULTS = None
        self.file_info_df = None
        self.transformed_resources_info_df = None
        self.get_transformed_device_info_df = None
        self.recon_results_df = None
        self.recon_pipelines_df = None

    def get_reconcilated_resources(self, resource_ref_lst):
        """Fetches how given FHIR resource reference ID reconciled into and by which pipeline"""
        RECON_RECORDS = []

        for resource_ref in resource_ref_lst:
            provenance_recon_url = Util.get_reconcilated_resources_url(FHIR_URL, resource_ref)
            resources = self.AUTH_QUERY.executeQuery(next_page_url=provenance_recon_url)
            recon_results={}
            recon_results['transformed_resource_url'] = resource_ref
            if resources['total'] != 0:
                recon_results['recon_into_resource'] = resources['entry'][0]['resource']['target'][0]['reference'] if  resources['entry'][0]['resource']['target'][0]['reference'] else "NA"
                recon_results['reconciled_by'] = resources['entry'][0]['resource']['agent'][0]['who']['reference'] if resources['entry'][0]['resource']['agent'][0]['who']['reference'] else "NA"
            RECON_RECORDS.append(recon_results)
        self.recon_results_df = pd.json_normalize(RECON_RECORDS)

    def populate_file_info(self):
        """Populates result with User provided information."""

        docref_url = Util.get_docref_resource_path_url(FHIR_URL, GCS_FILE_PATH)
        resources = self.AUTH_QUERY.executeQuery(next_page_url=docref_url)
        total_matched_resources = resources["total"]
        file_info = {}
        file_info['src_file_path'] = resources["entry"][0]['resource']['content'][0]['attachment']['url']
        file_info['src_file_docref_url'] = resources["entry"][0]['fullUrl']
        file_info['src_file_docref_id'] = resources["entry"][0]['resource']['id']
        file_info['src_file_info_meta_ts'] = resources["entry"][0]['resource']['meta']['lastUpdated']
        file_info['src_file_info_meta_versionId'] = resources["entry"][0]['resource']['meta']['versionId']

        # self.results.append(matched_resource_info)
        if GCS_FILE_PATH.strip().lower().endswith(".csv"):

            #Add details of csv file ingested into BQ table
            csv_file_docref_id = f"DocumentReference/{file_info['src_file_docref_id']}"

            docref_url_for_bq_tbl = Util.get_bq_table_docref_info(FHIR_URL, csv_file_docref_id)

            resources = self.AUTH_QUERY.executeQuery(next_page_url=docref_url_for_bq_tbl)

            file_info['bq_tbl_docref_id'] = resources['entry'][0]['resource']['target'][0]['reference']
            file_info['ingested_by_device_ref'] = resources['entry'][0]['resource']['agent'][0]['who']['reference']

            docref_url_for_bq = Util.get_docref_resources_url(FHIR_URL, str(file_info['bq_tbl_docref_id']).split("/")[-1])
            resources = self.AUTH_QUERY.executeQuery(next_page_url=docref_url_for_bq)

            file_info['tbl_name'] = resources['entry'][0]['resource']['content'][0]['attachment']['url']
            file_info['bq_tbl_id'] = resources["entry"][0]['resource']['id']

        df = pd.json_normalize(file_info)
        print(df.head())
        self.file_info_df = df

    def get_pipeline_info(self, device_ref_lst):
        """ Fetches Pipeline information using Device ID."""
        DEVICES = []

        for device_ref in device_ref_lst:
            device_info = {}
            if device_ref == "NA":
                device_info['device_id'] = "NA"
                device_info['transformed_job_name'] = "NA"
                device_info['transformed_job_start_ts'] = "NA"
            else:
                device_url = Util.get_device_id_url(FHIR_URL, str(device_ref).split("/")[-1])
                resources = self.AUTH_QUERY.executeQuery(next_page_url=device_url)
                device_info['device_id'] = device_ref
                if resources["total"] > 0:
                    for prop in resources["entry"][0]["resource"]["property"]:
                        if "JOB_NAME=" in str(prop["type"]["text"]).strip():
                            device_info['transformed_job_name'] = str(prop["type"]["text"]).split("=")[-1]
                        
                        elif re.match('PIPELINE_START_TIME=(.*)Z', str(prop["type"]["text"]).strip()):
                            device_info['transformed_job_start_ts'] = str(prop["type"]["text"]).split("=")[-1]

                        if "JOB_NAME" in device_info and "PIPELINE_START_TIME=" in device_info:
                            break
            DEVICES.append(device_info)
        df = pd.json_normalize(DEVICES)
        return df

    def get_transformed_df(self, resources):
        transformed_resources_info = []
        for resource in resources['entry']:
                temp_results = {}
                temp_results["transformation_provenance_url"] = resource["fullUrl"]
                temp_results["transformed_by_device_ref"] = resource['resource']['agent'][0]['who']['reference']
                temp_results["transformed_from_source_entity"] = resource["resource"]["entity"][0]["what"]["reference"]
                temp_results["transformation_provenance_id"] = resource["resource"]["id"]
                temp_results["transformed_resource_update_ts"] = resource["resource"]["meta"]["lastUpdated"]
                temp_results["transformed_resource_version"] = resource["resource"]["meta"]["versionId"]
                temp_results["transformed_resource_url"] = []
                for target_ref in resource["resource"]["target"]:
                    temp_results["transformed_resource_url"].append(target_ref['reference'])
                transformed_resources_info.append(temp_results)
        df = pd.json_normalize(transformed_resources_info).explode('transformed_resource_url')
        return df

    def get_csv_bq_ingestion_df(self):
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


    def get_transformed_fhir_fhir_resources(self):
        """Fetches the specific source records/data and respective pipeline of transformed into IFS FHIR resources."""
        
        for index, row in self.file_info_df.iterrows():
            if GCS_FILE_PATH.strip().lower().endswith(".csv"):
                #Add details of csv file ingested into BQ table
                provenance_url = Util.get_provenance_resource_path_url(FHIR_URL, row['bq_tbl_id'])
            else:
                provenance_url = Util.get_provenance_resource_path_url(FHIR_URL, row['src_file_docref_id'])
            
            #Fetches provenance resources using DocRef ID and produced harmonized resources Dataframe
            resources = self.AUTH_QUERY.executeQuery(next_page_url=provenance_url)
            self.transformed_resources_info_df = self.get_transformed_df(resources)

            #Joins harmonized resources Dataframe with source file information
            self.transformed_resources_info_df['tmp'] = 1
            self.file_info_df['tmp'] = 1
            self.RESULTS = pd.merge(self.transformed_resources_info_df, self.file_info_df, on=["tmp"])
            self.RESULTS.drop('tmp', axis=1)

            #Fetches harmonization pipeline information and craetes a Dataframe
            self.get_transformed_device_info_df = self.get_pipeline_info(self.RESULTS['transformed_by_device_ref'].unique())
            self.get_transformed_device_info_df = self.get_transformed_device_info_df.rename(columns={'device_id': 'transformed_device_id'})


            #Joins harmonization pipeline Dataframe with final Dataframe: RESULTS
            self.RESULTS = pd.merge(self.RESULTS, self.get_transformed_device_info_df, left_on='transformed_by_device_ref', right_on='transformed_device_id')

            if RECON_INFO:
                #Fetches respective reconciled final-fhir resource information and craetes a Dataframe
                self.get_reconcilated_resources(self.RESULTS['transformed_resource_url'].unique())

                #Joins reconciliation resources Dataframe with final Dataframe: RESULTS
                self.RESULTS = pd.merge(self.RESULTS, self.recon_results_df, left_on='transformed_resource_url', right_on='transformed_resource_url')

                #Fetches reconciliation pipeline information and craetes a Dataframe
                self.get_recon_device_info_df = self.get_pipeline_info(self.RESULTS['reconciled_by'].unique())
                self.get_recon_device_info_df = self.get_recon_device_info_df.rename(columns={'device_id': 'recon_device_id', 'transformed_job_name': 'recon_job_name',
                'transformed_job_start_ts' : 'recon_job_start_ts'})

                #Joins reconciliation pipeline Dataframe with final Dataframe: RESULTS
                self.RESULTS = pd.merge(self.RESULTS, self.get_recon_device_info_df, left_on='reconciled_by', right_on='recon_device_id')
            
            #Formatting final Dataframe: RESULTS
            self.RESULTS.drop('tmp', axis=1)

            #Write output Dataframe: RESULTS to Bigquery
            Util.handle_results(self.RESULTS)




def main():
    commons = Commons()
    print(f"processing OFS resources ...")
    commons.populate_file_info()
    commons.get_transformed_fhir_fhir_resources()


if __name__ == '__main__':
    main()    
