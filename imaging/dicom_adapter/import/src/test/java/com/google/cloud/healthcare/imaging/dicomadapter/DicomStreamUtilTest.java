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

import static com.google.common.truth.Truth.assertThat;

import com.google.api.client.util.IOUtils;
import com.google.cloud.healthcare.TestUtils;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import org.dcm4che3.data.Attributes;
import org.dcm4che3.data.Tag;
import org.dcm4che3.data.UID;
import org.dcm4che3.io.DicomInputStream;
import org.dcm4che3.net.PDVInputStream;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public final class DicomStreamUtilTest {
  // FakePDVInputStream is an implementation that reads from injected input stream, instead of
  // reading from network for DICOM.
  private class FakePDVInputStream extends PDVInputStream {
    private InputStream in;

    FakePDVInputStream(InputStream in) {
      this.in = in;
    }

    @Override
    public int read() throws IOException {
      return -1;
    }

    @Override
    public Attributes readDataset(String tsuid) throws IOException {
      return null;
    }

    @Override
    public void copyTo(OutputStream out, int length) {}

    @Override
    public void copyTo(OutputStream out) throws IOException {
      IOUtils.copy(this.in, out);
    }

    @Override
    public long skipAll() throws IOException {
      return -1;
    }
  }

  @Test
  public void testDicomStreamWithFileMetaHeader_correctTransferSyntax() throws Exception {
    FakePDVInputStream streamNoHeader =
        new FakePDVInputStream(TestUtils.streamDICOMStripHeaders(TestUtils.TEST_MR_FILE));
    String sopClassUID = UID.MRImageStorage;
    String sopInstanceUID = "1.0.0.0";
    String transferSyntax = UID.ExplicitVRLittleEndian;
    InputStream streamWithHeader =
        DicomStreamUtil.dicomStreamWithFileMetaHeader(
            sopInstanceUID, sopClassUID, transferSyntax, streamNoHeader);
    DicomInputStream dicomInputStream = new DicomInputStream(streamWithHeader);
    Attributes attrs = dicomInputStream.getFileMetaInformation();
    assertThat(attrs.getString(Tag.TransferSyntaxUID)).isEqualTo(transferSyntax);
  }
}
