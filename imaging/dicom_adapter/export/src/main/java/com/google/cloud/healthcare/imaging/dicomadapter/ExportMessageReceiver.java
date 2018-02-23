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

import com.google.cloud.pubsub.v1.AckReplyConsumer;
import com.google.cloud.pubsub.v1.MessageReceiver;
import com.google.pubsub.v1.PubsubMessage;

public class ExportMessageReceiver implements MessageReceiver {
  private DicomSender dicomSender;

  ExportMessageReceiver(DicomSender dicomSender) {
    this.dicomSender = dicomSender;
  }

  @Override
  public void receiveMessage(PubsubMessage message, AckReplyConsumer consumer) {
    try {
      dicomSender.send(message);
      consumer.ack();
    } catch (Exception e) {
      // TODO(b/73251955): Expose failures through cloud monitoring.
      e.printStackTrace();
      consumer.nack();
    }
  }
}
