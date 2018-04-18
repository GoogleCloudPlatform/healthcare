%{
Copyright 2018 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

> https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
%}

% Matlab BigQuery Access Demo Using JDBC
% This file serves as a quick demo on how to access Google BigQuery using a
% JDBC connector.
%
% Note: it is recommended that the Native BigQuery API is used for data
% analysis purposes, so please try using the Colab version or the R version
% of the tutorial as much as possible. Use the JDBC connection explained in
% this file only if your analysis can only be run in Matlab and not the
% alternatives.

% Step 1. Download BigQuery JDBC binary from
%     https://cloud.google.com/bigquery/partners/simba-drivers/
% Unpack the downloaded file, and locate the .jar files.

% Step 2. In Matlab, set the javaclasspath to include these .jar files,
% e.g.
javaclasspath({
    '/path/to/GoogleBigQueryJDBC42.jar',
    '/path/to/google-api-client-1.22.0.jar',
    '/path/to/google-api-services-bigquery-v2-rev355-1.22.0.jar',
    '/path/to/google-http-client-1.22.0.jar',
    '/path/to/google-http-client-jackson2-1.22.0.jar',
    '/path/to/google-oauth-client-1.22.0.jar',
    '/path/to/jackson-core-2.1.3.jar'})

% Step 3. Set up JDBC connection to BigQuery, and issue query using tall()
% function, as explained in
% https://www.mathworks.com/help/database/examples/import-large-data-using-a-databasedatastore-and-tall-array.html
bq_driver = 'com.simba.googlebigquery.jdbc42.Driver'

% Build the connection URL by following
% https://www.simba.com/products/BigQuery/doc/JDBC_InstallGuide/content/jdbc/bq/using/connectionurl.htm
% For example,
bq_url = 'jdbc:bigquery://https://www.googleapis.com/bigquery/v2:443;ProjectId=datathon-datasets;OAuthType=1;'

% Leave user and password empty.
conn = database('BigQuery', '', '', bq_driver, bq_url)

query = 'SELECT COUNT(*) AS num_records FROM mimic_demo.icustays'

% Important: When executing this step, an "Authorization Code Required"
% window will pop up, with a URL provided in an edit box. Copy this URL to
% a browser window, and open the link. You will be directed to an
% authentication page. After authenticating yourself and clicking "Allow"
% in the "BigQuery Client Tools wants to access your Google Account"
% window, an authentication code string will be generated and displayed in
% the browser window. Please copy the authentication code back to the edit
% box in Matlab to continue.
dbds = databaseDatastore(conn, query)
result = tall(dbds)

% Step 4. Process the query result in Matlab.
disp(result)

% Step 5. Close the connection.
close(conn)
