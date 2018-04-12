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
import com.google.api.core.ApiService;
import com.google.auth.http.HttpCredentialsAdapter;
import com.google.auth.oauth2.GoogleCredentials;
import com.google.cloud.healthcare.DicomWebClient;
import com.google.cloud.healthcare.LogUtil;
import com.google.cloud.pubsub.v1.Subscriber;
import com.google.common.util.concurrent.MoreExecutors;
import com.google.pubsub.v1.SubscriptionName;
import java.io.IOException;
import java.security.GeneralSecurityException;
import java.util.Arrays;
import org.dcm4che3.net.ApplicationEntity;
import org.dcm4che3.net.Connection;

public class ExportAdapter {
  public static HttpRequestFactory createHttpRequestFactory(GoogleCredentials credentials) {
    if (credentials == null) {
      return new NetHttpTransport().createRequestFactory();
    }
    return new NetHttpTransport().createRequestFactory(new HttpCredentialsAdapter(credentials));
  }

  public static void main(String[] args) throws IOException, GeneralSecurityException {
    Flags flags = new Flags();
    JCommander jCommander = new JCommander(flags);
    jCommander.parse(args);

    // Adjust logging.
    if (flags.verbose) {
      LogUtil.Log4jToStdout();
    }

    // DicomWeb client for source of DICOM.
    GoogleCredentials credentials = GoogleCredentials.getApplicationDefault();
    if (!flags.oauthScopes.isEmpty()) {
      credentials = credentials.createScoped(Arrays.asList(flags.oauthScopes.split(",")));
    }
    DicomWebClient dicomWebClient =
        new DicomWebClient(createHttpRequestFactory(credentials), flags.dicomwebAddr);

    // Use either C-STORE or STOW-RS to send DICOM, based on flags.
    boolean isStowRs = !flags.peerDicomwebAddr.isEmpty() && !flags.peerDicomwebStowPath.isEmpty();
    boolean isCStore =
        !flags.peerDimseAET.isEmpty() && !flags.peerDimseIP.isEmpty() && flags.peerDimsePort != 0;
    DicomSender dicomSender = null;
    if (isStowRs && isCStore) {
      System.err.println("Both C-STORE and STOW-RS flags should not be specified.");
      System.exit(1);
    } else if (isStowRs) {
      // STOW-RS sender.
      //
      // DicomWeb client for sink of DICOM.
      // Use application default credentials for HTTP requests of DICOM sink, if specified by flag.
      HttpRequestFactory requestFactory = null;
      if (flags.useGcpApplicationDefaultCredentials) {
        requestFactory = createHttpRequestFactory(credentials);
      } else {
        requestFactory = createHttpRequestFactory(null);
      }
      DicomWebClient exportDicomWebClient =
          new DicomWebClient(requestFactory, flags.peerDicomwebAddr);
      dicomSender =
          new StowRsSender(dicomWebClient, exportDicomWebClient, flags.peerDicomwebStowPath);
      System.out.printf(
          "Export adapter set-up to export via STOW-RS to address: %s, path: %s\n",
          flags.peerDicomwebAddr, flags.peerDicomwebStowPath);
    } else if (isCStore) {
      // C-Store sender.
      //
      // DIMSE application entity.
      ApplicationEntity applicationEntity = new ApplicationEntity("EXPORTADAPTER");
      Connection conn = new Connection();
      DeviceUtil.createClientDevice(applicationEntity, conn);
      applicationEntity.addConnection(conn);
      dicomSender =
          new CStoreSender(
              applicationEntity,
              flags.peerDimseAET,
              flags.peerDimseIP,
              flags.peerDimsePort,
              dicomWebClient);
      System.out.printf(
          "Export adapter set-up to export via C-STORE to AET: %s, IP: %s, Port: %d\n",
          flags.peerDimseAET, flags.peerDimseIP, flags.peerDimsePort);
    } else {
      System.err.println("Neither C-STORE nor STOW-RS flags have been specified.");
      System.exit(1);
    }

    Subscriber subscriber = null;
    try {
      SubscriptionName subscriptionName =
          SubscriptionName.of(Flags.projectId, Flags.subscriptionId);
      subscriber =
          Subscriber.newBuilder(subscriptionName, new ExportMessageReceiver(dicomSender)).build();
      subscriber.addListener(
          new Subscriber.Listener() {
            @Override
            public void failed(Subscriber.State from, Throwable failure) {
              System.err.println(failure);
            }
          },
          MoreExecutors.directExecutor());
      ApiService service = subscriber.startAsync();
      service.awaitRunning();

      System.out.println("Pubsub listener up and running.");
      service.awaitTerminated();
    } finally {
      if (subscriber != null) {
        subscriber.stopAsync();
      }
    }
  }
}
