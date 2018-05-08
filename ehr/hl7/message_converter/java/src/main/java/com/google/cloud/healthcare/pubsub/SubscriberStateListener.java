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

package com.google.cloud.healthcare.pubsub;

import com.google.api.core.ApiService.Listener;
import com.google.api.core.ApiService.State;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Listens for state changes of a {@link com.google.cloud.pubsub.v1.Subscriber}.
 */
public class SubscriberStateListener extends Listener {
  private static final Logger LOGGER =
      Logger.getLogger(SubscriberStateListener.class.getCanonicalName());

  @Override
  public void failed(State from, Throwable failure) {
    LOGGER.log(Level.SEVERE, "From state: " + from.name(), failure);
  }

  @Override
  public void running() {
    LOGGER.log(Level.INFO, "Running");
  }

  @Override
  public void starting() {
    LOGGER.log(Level.INFO, "Starting");
  }

  @Override
  public void stopping(State from) {
    LOGGER.log(Level.INFO, "Stopping from state: " + from.name());
  }

  @Override
  public void terminated(State from) {
    LOGGER.log(Level.INFO, "Terminated from state: " + from.name());
  }
}
