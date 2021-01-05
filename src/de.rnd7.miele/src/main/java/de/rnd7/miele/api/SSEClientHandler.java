package de.rnd7.miele.api;

import org.apache.client.sse.Event;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.json.JSONObject;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.BlockingQueue;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;

class SSEClientHandler {
    private static final Logger LOGGER = LoggerFactory.getLogger(SSEClientHandler.class);
    public static final int TIMEOUT = 1_000;
    public static final int MAX_RETRY_CTR = 10_000 / TIMEOUT;

    private final BlockingQueue<Event> events;
    private final CloseableHttpAsyncClient asyncClient;
    private final Consumer<MieleDevice> consumer;

    public SSEClientHandler(final BlockingQueue<Event> events, final CloseableHttpAsyncClient asyncClient, final Consumer<MieleDevice> consumer) {
        this.events = events;
        this.asyncClient = asyncClient;
        this.consumer = consumer;
    }

    public void run() throws Exception {
        int noMessageCtr = 0;

        while (true) {
            final Event event = events.poll(TIMEOUT, TimeUnit.MILLISECONDS);
            if (event != null) {
                noMessageCtr = 0;

                handleEvent(event);
            }
            else if (noMessageCtr >= MAX_RETRY_CTR || !asyncClient.isRunning()) {
                // No message for more than 10s (not even ping) - try reconnect.
                asyncClient.close();
                events.stream().close();
                break;
            }
            else {
                noMessageCtr++;
            }
        }
    }

    private void handleEvent(final Event event) {
        if (event.getEvent().equals("devices")) {
            handleDevices(event);
        } else if (event.getEvent().equals("ping")) {
            handlePing();
        }
    }

    private void handleDevices(final Event event) {
        final JSONObject devices = new JSONObject(event.getData());

        devices.keySet()
            .stream()
            .map(id -> new MieleDevice(id, devices.getJSONObject(id)))
            .forEach(consumer::accept);
    }

    private void handlePing() {
        LOGGER.debug(".");
    }

}
