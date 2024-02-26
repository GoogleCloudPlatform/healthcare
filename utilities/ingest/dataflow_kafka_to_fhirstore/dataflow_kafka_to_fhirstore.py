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


##################################
# Author: Devansh Modi           #
##################################

#Main Dataflow MODULE :- fetching from PubSub, cleaning, filtering, transforming and posting to Cloud Healthcare API FHIR store

#Importing Necessary Libraries
from google.cloud import secretmanager
import apache_beam as beam
from apache_beam import pvalue
from apache_beam import io
from apache_beam.io.kafka import ReadFromKafka
from apache_beam.options.pipeline_options import PipelineOptions
import json
import logging
import argparse
from datetime import datetime, timedelta
import google.auth
import google.auth.transport.requests
import requests
import base64
import uuid
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry
import configparser
import time

   
#Below class is used to POST and GET FHIR resources from HealthCare API FHIR store (We will use Patient FHIR resource as an example)
class hcapi_fhir_store:
    def __init__(self, args:dict):
        self.hcapi_project_id = str(args['hcapi_project_id'])
        self.hcapi_version = str(args['hcapi_version'])
        self.hcapi_location = str(args['hcapi_location'])
        self.hcapi_dataset = str(args['hcapi_dataset'])
        self.hcapi_get_fhir_store = str(args['hcapi_get_fhir_store'])
        self.hcapi_post_fhir_store = str(args['hcapi_post_fhir_store'])
      

    def google_api_headers(self):
        """ This function gets access tokens and authorizations\
            to access cloud healthcare API Fhir store"""
        creds, project = google.auth.default()
        auth_req = google.auth.transport.requests.Request()
        creds.refresh(auth_req)
        authToken = creds.token
        authHeaders = {
            "Authorization": f"Bearer {authToken}",
            "Prefer": "handling=strict"
            }
        return authHeaders

    def hcapi_dataset_url(self, version: str =None, project: str=None, location: str=None, dataset: str=None):
        """ This function creates base hcapi dataset url and returns it """
        """
        :param str version: HCAPI version from arguments passed
        :param str project: HCAPI project from arguments passed
        :param str location: HCAPI location from arguments passed
        :param str dataset: HCAPI dataset from arguments passed
        :return str hcapiDatasetURL: HCAPI dataset url build
        """
        base = "https://healthcare.googleapis.com"
        version = self.hcapi_version
        project = self.hcapi_project_id
        location = self.hcapi_location
        dataset = self.hcapi_dataset
        hcapiDatasetURL = f"{base}/{version}/projects/{project}/locations/{location}/datasets/{dataset}"
        return hcapiDatasetURL


    def hcapi_fhir_url(self, version: str =None, project: str=None, location: str=None, dataset: str=None,store: str= None):
        """ This function creates base hcapi FHIR url """
        """
        :param str version: HCAPI version from arguments passed
        :param str project: HCAPI project from arguments passed
        :param str location: HCAPI location from arguments passed
        :param str dataset: HCAPI dataset from arguments passed
        :param str store: HCAPI FHIR store from arguments passed
        :return str hcapiFHIRUrl: HCAPI dataset url build
        """
        hcapiDatasetURL = self.hcapi_dataset_url(version=version, project=project,
        location=location, dataset=dataset)
        if store is None:
            raise Exception("No FHIR store specified")
        else:
            store = store
        hcapiFHIRUrl = f'{hcapiDatasetURL}/fhirStores/{store}/fhir'
        return hcapiFHIRUrl
    
    def createRequestSession(self):
        """Creating request session to try GET/POST requests using below force list"""
        reqSession = requests.Session()
        retries = Retry(total=3,
                        backoff_factor=2,
                        status_forcelist=[429, 500, 502, 503, 504, 400, 404, 401])

        reqSession.mount('http://', HTTPAdapter(max_retries=retries))
        return reqSession

    def hcapi_get(self, url:str):
        """ This function to get FHIR resource from FHIR store specified """
        """
        :param str url: HCAPI FHIR URL to fetch data
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        try:
            responseFlag = "success"
            reqSession = self.createRequestSession()
            logging.info("creating headers with new token for HCAPI Fhir store GET request")
            creds, project = google.auth.default()
            auth_req = google.auth.transport.requests.Request()
            creds.refresh(auth_req)
            authToken = creds.token
            authHeaders = {
                "Authorization": f"Bearer {authToken}",
                "Prefer": "handling=strict"
            }
            logging.info("Fetching FHIR resource from HCAPI FHIR store")
            response = reqSession.get(url, headers=authHeaders,timeout=30)
            response.raise_for_status()
            responseJSON = response.json()
        except Exception as error:
            reqSession.close()
            responseFlag = "fail"
            logging.error("process to get FHIR resource failed")
            errorMessage = "Process to fetch FHIR resource from HCAPI FHIR store failed due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            responseJSON = dict()
            responseJSON['errorMessage'] = errorMessage
            return responseJSON,responseFlag
        else:
            reqSession.close()
            return responseJSON,responseFlag

    def hcapi_post(self, url:str, data:str):
        """ This function to POST FHIR resource from FHIR store specified """
        """
        :param str url: HCAPI FHIR store URL to fetch data
        :param str data: JSON payload string
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        try:
            responseFlag = "success"
            logging.info("creating headers with new token for HCAPI FHIR store POST request")
            creds, project = google.auth.default()
            auth_req = google.auth.transport.requests.Request()
            creds.refresh(auth_req)
            authToken = creds.token
            authHeaders = {
                "Authorization": f"Bearer {authToken}",
                "Prefer": "handling=strict"
            }
            logging.info("Posting FHIR resource to HCAPI FHIR store")
            payload_data =json.loads(data)
            reqSession = self.createRequestSession()
            response = requests.post(url, headers=authHeaders, json=payload_data,timeout=30)
            response.raise_for_status()
            responseJSON = response.json()
        except Exception as error:
            reqSession.close()
            logging.error("process to POST FHIR resource failed")
            errorMessage = "Process to POST FHIR Resource to HCAPI FHIR store failed due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            responseJSON = dict()
            responseJSON['errorMessage'] = errorMessage
            responseFlag = "fail"
            return responseJSON,responseFlag
        else:
            reqSession.close()
            return responseJSON,responseFlag

    def post_fhir_message(self, resourceType:str,payload:str):
        """ Function to post messages to FHIR store """
        """
        :param str resourceType: FHIR Resource to post data
        :param str payload: FHIR Resource as string
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        resourceType=str(resourceType)
        url = self.hcapi_fhir_url(store=self.hcapi_post_fhir_store)+f'/{resourceType}'
        postResponse,responseFlag = self.hcapi_post(url,payload)
        return postResponse,responseFlag

    def get_fhir_message(self, url_path:str):
        """ Function to fetch FHIR resources from HCAPI FHIR store"""
        """
        :param str url_path: FHIR resource URL to get FHIR resource from HCAPI FHIR store
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        url_path=str(url_path)
        url = self.hcapi_fhir_url(store=self.hcapi_get_fhir_store)+f'/{url_path}'
        responseJSON, responseFlag = self.hcapi_get(url)
        return responseJSON, responseFlag


class transformations:
    """Generic class to build multiple FHIR resource as per business requirements.
        For example, we are building Patient FHIR resource """
    
    def buildIdentifier(message:dict, messageKey:str):
        """Function to build mulitple identifiers for a given FHIR resource"""
        """
        :param dict message: raw message
        :param str messageKey: identifier value example SSN, ID, etc.
        :return dict identifier: Identifier dictionary with system and value as per US Core
        """
        identifier= dict()
        identifier['system'] = str(messageKey)
        identifier['value'] = message[messageKey] if message[messageKey] else "N/A"
        return identifier

    @staticmethod
    def buildPatientResource(message:dict):
        """Function to create Patient FHIR resource from JSON payload receieved via member-details kafka topic """
        """
        :param dict message: raw JSON payload dict
        :return dict patientResourceJSON: patient FHIR resource JSON
        :return str processFlag: process flag to indicate if the process failed or passed
        """
        try:
            patientResource = dict()
            patientResource['resourceType']="Patient"
            indentifier = list()
            indentifier.append(transformations.buildIdentifier(message,'SSN'))
            patientResource['identifier'] = indentifier
            patientResource['Id'] = message['Id'] if message['Id'] else "N/A"
            patientName = dict()
            patientName['use'] = 'official'
            patientName['family'] = message['LAST'] if message['LAST'] else "N/A"
            patientName['given'] =[message['FIRST'] if message['FIRST'] else "N/A"]
            patientResource['name'] = [patientName]
            patientResource['birthDate'] = message['BIRTHDATE'] if message['BIRTHDATE'] else "N/A"
            patientResource['gender'] = message['GENDER'] if message['GENDER'] else "N/A"
            address = dict()
            address['use'] = 'home'
            address['type'] = 'physical'
            address['line'] = [message['ADDRESS'] if message['ADDRESS'] else "N/A"]
            address['district'] = message['COUNTY'] if message['COUNTY'] else "N/A"
            address['city'] = message['CITY'] if message['CITY'] else "N/A"
            address['state'] = message['STATE'] if message['STATE'] else "N/A"
            address['country'] = message['COUNTRY'] if message['COUNTRY'] else "N/A"
            patientResource['address'] = [address]
            telecom = dict()
            telecom['use'] = 'home'
            telecom['system'] = 'email'
            telecom['value'] = message['EMAIL'] if message['EMAIL'] else "N/A"
            patientResource['telecom'] = [telecom]
            patientResourceJSON = json.loads(json.dumps(patientResource,separators=(',', ':')))
            logging.info("process to build patient FHIR resource completed")
            
            # printing transform Patient FHIR resource for monitoring purposes.
            # Please remove in production as it is not recommnended to print PHI data as per HIPAA compliance
            logging.info(f"printing transform Patient FHIR resource for monitoring purposes {json.dumps(patientResourceJSON)}")
            processFlag = "success"
        except Exception as error :
            logging.warning("process to build Patient FHIR Resource failed due to : {}".format(error, exc_info=True))
            errorMessage = "Process to build Patient FHIR Resource failed due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            rawMessage = dict()
            rawMessage['rawMessage'] = message
            rawMessage['errorMessage'] = errorMessage
            processFlag = "fail"
            return rawMessage,processFlag
        else:
            return patientResourceJSON,processFlag
    

class buildPatientResource(beam.DoFn):
    """ Beam DoFn to transform raw pateint data and create Patient FHIR resource"""
    #beam output tag to mark transformed element as pass and fail
    failTag = "failRecord"
    passTag = "passRecord"

    def process(self,element):
        """
         process to fetch, clean careplan, transform data and return build careplan
        :param recordType element: PCollection of message from filterKafkaRecord DoFn
        :return: cleaned, transformed CareaPlan
        :rtype: dict
        """
        jsonElement = json.loads(json.dumps(element,separators=(',', ':')))
        messageValue = json.loads(jsonElement['message'])
        patientResourceJSON,patientResourceFlag = transformations.buildPatientResource(messageValue)
        logging.info("transforming raw JSON to create Patient FHIR resource")
        if patientResourceFlag == "success":
            logging.info("Successfully transformed payload to Patient FHIR resource from Member Data topic")
            yield pvalue.TaggedOutput(self.passTag,patientResourceJSON)
        else:
            logging.error("aborting the process to build Patient FHIR Resource as patientResourceFlag is failed")
            yield pvalue.TaggedOutput(self.failTag,patientResourceJSON)
        

class filterKafkaRecord(beam.DoFn):
    """ Beam Do Fn to transform and filter kafka raw message for further archiving & transformation """
    #beam output tag to mark transformed element as pass and fail
    failTag = "failRecord"

    """ Output tag to determine which resource type the raw data belongs to.\
        This will help in determining transformations specific to that resource type
        For the purpose of this pipeline we will use patientResource and Careplan Resource is shown as example
        careplanResourceTag = "careplanResource"    
        """
    patientResourceTag = "patientResource"
    
    def process(self, element):
        """
        :param recordType element: PCollection of message from Kafka topic
        :return: transformed and filtered raw message as per Nicotine Cessation Program Requirements
        :rtype: dict
        """
        try:
            logging.info("transforming and filtering raw kafka record")

            #to do remove this logging only meant for testing purpose
            logging.info("printing raw record from kafka value --> {}".format(str(element)))
            
            # Converting bytes record from Kafka to a dictionary
            transformedPayload = dict()
            messageTimestamp = datetime.now().strftime("%Y-%m-%dT%H:%M:%S")
            newElement = element
            rawMessage = json.loads(newElement[1].decode("utf-8"))
            logging.info("printing raw message value --> {}".format(json.dumps(rawMessage)))
            transformedPayload['message'] = json.dumps(rawMessage,separators=(',', ':'))
            transformedPayload['timestamp'] = messageTimestamp
            yield pvalue.TaggedOutput(self.patientResourceTag,transformedPayload)

            """ We can also write necessary filters to determine whether this is a Patient Resource or matches the given business requirement
            Considering following raw JSON recieved via topic
                {"ADDRESS":"978 Turcotte Overpass","BIRTHDATE":"2004-07-04","COUNTRY":"US","CITY":"Gill","COUNTY":"Franklin County","DRIVERS":"",\
                "ETHNICITY":"nonhispanic","FIRST":"Gidget756","GENDER":"Female","HEALTHCARE_COVERAGE":3311.4399999999996,"HEALTHCARE_EXPENSES":404742.42,\
                "Id":"7992bf94-feee-4728-9187-2c911df2819b","LAST":"Rempel203","LAT":42.63299146919133,"LON":-72.5251235198123,"MAIDEN":"","MARITAL":"",\
                "EMAIL":"sampleemail@.yahoo.com","PREFIX":"","RACE":"white","SSN":"999-67-4045","STATE":"Massachusetts","SUFFIX":"","PROGRAM_NAME":"NEW_ENROLLMENT"}

            One can write filters to check whether the country code and program name matches or note as showing in the given if..else block
            rawMessageKeys = sorted(list(rawMessage.keys()))
            if rawMessage['COUNTRY'] == "US" and rawMessage['PROGRAM_NAME']== "NEW_ENROLLMENT":
                transformedPayload['new_member'] = "YES"
                transformedPayload['message'] = json.dumps(rawMessage,separators=(',', ':'))
                transformedPayload['timestamp'] = messageTimestamp
                yield pvalue.TaggedOutput(self.patientResourceTag,transformedPayload)
            else:
                logging.info("skipping to transform data as PROGRAM_NAME/COUNTRY key not found or data validation doesn't meet the given criteria") 
            """
        except Exception as error :
            logging.error("process to convert Kafka record failed due to : {}".format(error, exc_info=True))
            errorMessage = f"Process to Convert and Filter Kafka record failed due to : {str(type(error).__name__)} --> {str(error)}"
            transformedPayload = dict()
            transformedPayload['errorMessage'] = errorMessage
            transformedPayload['rawMessage']= str(element)
            yield pvalue.TaggedOutput(self.failTag,transformedPayload)      
        


class writeToGCS(beam.DoFn):
    """ Class to write raw and errored messages to GCS """
    def process(self, element, bucket, messageType, topic=None):
        """
         write raw / error messages to respective bucket
        :param str element: PCollection of message from convert kafka record function
        :param str bucket: bucket name
        :param str messageType: message type raw or error message
        :return: None message written to cloud storage
        :rtype: None
        """
        #TO DO below logging only for testing purpose delete when deploying to prod
        logging.info("printing raw element : {}".format(str(json.dumps(element))))
        
        if messageType :
            folderName= str(messageType)+"_messages"
        else:
            folderName= "unidentified"
        
        #converting incoming element to JSON
        newElement = json.loads(json.dumps(element,separators=(',', ':')))
        if topic:
            topicName = str(topic)
        elif 'topic' in newElement.keys():
            topicName = str(newElement['topic'])
        else:
            logging.info("topic name not mentioned or not present in topic")
        
        if topicName:
            if folderName and bucket:
                strJsonElement = str(newElement).encode('utf-8')
                #Building GCS path based on current timestamp
                currentTimestamp = datetime.now().strftime("%Y-%m-%d-%H%M%S")
                gcsPath = f"gs://{bucket}/{folderName}/{topicName}/{str(currentTimestamp[:10])}/{str(currentTimestamp.replace('-',''))}.json"
                file = io.gcsio.GcsIO().open(filename=gcsPath, mode='w', mime_type='application/json')
                write_file = file.write(strJsonElement)
                logging.info("successfully written message to cloud storage at :- {}".format(gcsPath))                
            else:
                logging.error("foldername and bucket not mentioned")
        else:
            logging.error("topic name not mentioned or not present in topic")
            

class postToHCAPI(beam.DoFn):
    """ Class to read file, clean and seperate messgaes based on MSH"""
    #beam output tag to mark transformed element as pass and fail
    failTag = "failRecord"

    def process(self, element,resourceType):
        logging.info("starting to prepare and post messages")
        dataJson = json.dumps(element,separators=(',', ':'))
        postResponseJSON,postResponseFlag  = hcapi.post_fhir_message(resourceType=str(resourceType),payload=dataJson)
        if postResponseFlag == "success":
            logging.info("successfully written FHIR Resource to given FHIR store")
        elif postResponseFlag == "fail":
            logging.error("failed to write (POST) transformed FHIR Resource to given FHIR store")
            failPostMessage = json.loads(json.dumps(element,separators=(',', ':')))
            failPostMessage['postHCAPIErrorMessage'] = postResponseJSON
            yield pvalue.TaggedOutput(self.failTag,failPostMessage)
            

def run(args_dict, beam_args,argv=None):
    beam_pipeline_options = PipelineOptions(beam_args,streaming=True,save_main_session=True,sdk_location="container")
    gcs_archive_bucket= str(args_dict['gcs_archive_bucket'])
    gcs_error_bucket= str(args_dict['gcs_error_bucket'])
    topic_list = args_dict['topics'].split(",")
    
    # only use if using the pipeline in batch mode
    # offset_days = int(args_dict['offset_days'])

    #To be used in PROD to fetch Kafka connection details from Secret manager preferred approach
    client_secret = secretmanager.SecretManagerServiceClient()
    response_kafka = client_secret.access_secret_version(name=str(args_dict['kafka_secret_resource_name']))
    kafka_secrets = json.loads(response_kafka.payload.data.decode("UTF-8"))
    bootstrap_servers = str(kafka_secrets['bootstrap_servers'])
   
    #If using confluent Kafka you will need API Key and API secret to login to Bootstrap servers
    # Example of secrets server_credentials_kafka = json.loads('{"bootstrap_servers": 'xx.xxx.x.xx:port',"apiKey":"xyz","apiSecret":"abc"}')
    # apiKey = kafka_secrets['apiKey']
    # apiSecret = kafka_secrets['apiSecret']

    """ 
        Creating Kafka consumer configurations properties for connecting to Kafka you can add more \ 
        based on your requirements based on Kafka cluster you are connecting to and which is allowed by beam
    """
    KafkaConsumerConfig = dict()
    KafkaConsumerConfig['bootstrap.servers']=str(bootstrap_servers)
    KafkaConsumerConfig['group.id']="test-kafka-to-hcapi"
    KafkaConsumerConfig['auto.offset.reset']="earliest"
    KafkaConsumerConfig['session.timeout.ms']="90000"
    KafkaConsumerConfig['fetch.max.wait.ms']="500"
    KafkaConsumerConfig['request.timeout.ms']="120000"

    # Update Kafka consumer config to the following if connecting to confluent Kafka Cloud via api key and api secret
    # KafkaConsumerConfig['sasl.mechanism']='PLAIN'
    # KafkaConsumerConfig['security.protocol']='SASL_SSL'
    # KafkaConsumerConfig['sasl.jaas.config']=f"""org.apache.kafka.common.security.plain.PlainLoginModule required username=\"{apiKey}\" password=\"{apiSecret}\";"""

    with beam.Pipeline(options=beam_pipeline_options) as pipeline:
        consumeRawMessage = (
            pipeline
            | "ReadFromKafka" >> ReadFromKafka(
                consumer_config=KafkaConsumerConfig,
                topics=topic_list,
                key_deserializer = 'org.apache.kafka.common.serialization.ByteArrayDeserializer' , 
                value_deserializer = 'org.apache.kafka.common.serialization.ByteArrayDeserializer'

                # Enable the option to run pipeline in batch mode as per schedule or adhoc basis
                # start_read_time = int((datetime.now()- timedelta(days=offset_days)).strftime("%s")) * 1000
                )
            | "FilterConvertKafkaRecord" >> beam.ParDo(filterKafkaRecord()).with_outputs()
        )
        writeFailRawMsgToGCS = (
            consumeRawMessage.failRecord
            | "WriteFailedConvertMsgToGCS" >> beam.ParDo(writeToGCS(),topic="fail_filter_convert_messages",bucket=gcs_archive_bucket,messageType="error")
        )

        # We can tag individual type of FHIR resource and archive it parallely from load balancing perspective and to make efficient use of resources. 
        archivePatientResource = (
            consumeRawMessage.patientResource
            | "archiveMemberDetailsToGCS" >> beam.ParDo(writeToGCS(),topic="member_raw_details",bucket=gcs_archive_bucket,messageType="raw")
        )
        # #patient FHIR Resource
        buildPatient = (
            consumeRawMessage.patientResource
            | "buildPatientResource" >> beam.ParDo(buildPatientResource()).with_outputs()
        )
        writeFailPatientMsgToGCS = (
            buildPatient.failRecord
            | "writeFailedPatientMsgToGCS" >> beam.ParDo(writeToGCS(),topic="fail_patient_transform_messages",bucket=gcs_error_bucket,messageType="error")
        )
        postPatientMsgToHCAPI = (
            buildPatient.passRecord 
            | "postPatientToHCAPI" >> beam.ParDo(postToHCAPI(),resourceType="Patient").with_outputs()
        )
        writeFailPostPatientMsgToGCS = (
            postPatientMsgToHCAPI.failRecord
            | "writeFailPatientPOSTMsgToGCS" >> beam.ParDo(writeToGCS(),topic="fail_hcapi_post_messages",bucket=gcs_error_bucket,messageType="error")
        )

if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG, format='%(asctime)s :: %(levelname)-8s :: [%(filename)s:%(lineno)d] :: %(message)s')
    parser = argparse.ArgumentParser()
    # parser.add_argument('--bootstrap_servers',required=True, help='string of bootstrap servers to connect to kafka')
    parser.add_argument('--topics', required=True, help='string of single or multiple topics seperated by comma')
    parser.add_argument('--kafka_secret_resource_name', required=True, help='secret resource name to get kafka secrets')
    parser.add_argument('--gcs_archive_bucket', required=True, help='gcs bucket to archive raw messages after consumption')
    parser.add_argument('--gcs_error_bucket', required=True, help='gcs bucket to store error messages')
    parser.add_argument('--hcapi_project_id', required=True, help='HCAPI project ID')
    parser.add_argument('--hcapi_dataset', required=True, help='HCAPI dataset')
    parser.add_argument('--hcapi_version', required=True, help='HCAPI Version')
    parser.add_argument('--hcapi_location', required=True, help='HCAPI location')
    parser.add_argument('--hcapi_get_fhir_store', required=True, help='HCAPI FHIR Store to fetch FHIR resources')
    parser.add_argument('--hcapi_post_fhir_store', required=True, help='HCAPI POST FHIR Store')
    parser.add_argument('--offset_days', required=False, help='offset days to go back and pull data from Kafka topics only needed if running in Batch mode')
    args, beam_args = parser.parse_known_args()
    args_dict = vars(args)
    hcapi= hcapi_fhir_store(args_dict)
    transform = transformations()
    run(args_dict=args_dict,beam_args=beam_args)
