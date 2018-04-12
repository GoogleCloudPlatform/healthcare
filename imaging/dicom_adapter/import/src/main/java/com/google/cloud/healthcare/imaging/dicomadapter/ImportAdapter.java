// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package com.google.cloud.healthcare.imaging.dicomadapter;

import com.beust.jcommander.JCommander;
import com.google.api.client.http.HttpRequestFactory;
import com.google.api.client.http.javanet.NetHttpTransport;
import com.google.auth.http.HttpCredentialsAdapter;
import com.google.auth.oauth2.GoogleCredentials;
import com.google.cloud.healthcare.LogUtil;
import java.io.IOException;
import java.security.GeneralSecurityException;
import java.util.Arrays;
import org.dcm4che3.net.Device;
import org.dcm4che3.net.service.BasicCEchoSCP;
import org.dcm4che3.net.service.DicomServiceRegistry;

public class ImportAdapter {
  public static void main(String[] args) throws IOException, GeneralSecurityException {
    Flags flags = new Flags();
    JCommander jCommander = new JCommander(flags);
    jCommander.parse(args);

    // Adjust logging.
    if (flags.verbose) {
      LogUtil.Log4jToStdout();
    }

    // Credentials, use the default service credentials.
    GoogleCredentials credentials = GoogleCredentials.getApplicationDefault();
    if (!flags.oauthScopes.isEmpty()) {
      credentials = credentials.createScoped(Arrays.asList(flags.oauthScopes.split(",")));
    }

    // Dicom service handlers.
    DicomServiceRegistry serviceRegistry = new DicomServiceRegistry();

    // Handle C-ECHO (all nodes which accept associations must support this).
    serviceRegistry.addDicomService(new BasicCEchoSCP());

    // Handle C-STORE.
    HttpRequestFactory requestFactory =
        new NetHttpTransport().createRequestFactory(new HttpCredentialsAdapter(credentials));
    CStoreService cStoreService =
        new CStoreService(flags.dicomwebAddr, flags.dicomwebStowPath, requestFactory);
    serviceRegistry.addDicomService(cStoreService);

    Device device = DeviceUtil.createServerDevice(flags.dimseAET, flags.dimsePort, serviceRegistry);
    device.bindConnections();
  }
}
