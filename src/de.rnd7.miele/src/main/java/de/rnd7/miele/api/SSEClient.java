package de.rnd7.miele.api;

import org.apache.client.sse.ApacheHttpSseClient;
import org.apache.client.sse.Event;
import org.apache.client.sse.SseRequest;
import org.apache.client.sse.SseResponse;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.impl.nio.client.HttpAsyncClients;
import org.json.JSONObject;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;

public class SSEClient {
    private static final Logger LOGGER = LoggerFactory.getLogger(SSEClient.class);
    public static final int TIMEOUT = 1000;
    public static final int MAX_RETRY_CTR = 10_000 / TIMEOUT;
    private final ExecutorService executor = Executors.newFixedThreadPool(1);
    private CloseableHttpAsyncClient asyncClient;

    BlockingQueue<Event> subscribe(final MieleAPI api) throws Exception {
        LOGGER.debug("Subscribe SSE");
        final String token = api.getToken().getAccessToken();
        asyncClient = HttpAsyncClients.createDefault();
        asyncClient.start();
        final SseRequest request = new SseRequest("https://api.mcs3.miele.com/v1/devices/all/events");
        request.setHeader("Accept-Language", "en-GB");
        request.setHeader("Authorization", "Bearer " + token);
        final ApacheHttpSseClient sseClient = new ApacheHttpSseClient(asyncClient, executor);

        final Future<SseResponse> future = sseClient.execute(request);

        final SseResponse sseResponse = future.get(10, TimeUnit.SECONDS);
        return sseResponse.getEntity().getEvents();
    }

    void shutdown() {
        closeClient();

        executor.shutdown();
    }

    void closeClient() {
        if (asyncClient != null) {
            try {
                asyncClient.close();
            } catch (IOException e) {
                LOGGER.error(e.getMessage(), e);
            }
        }
    }

    boolean isRunning() {
        return asyncClient.isRunning();
    }

    public void start(final MieleAPI api, final Consumer<MieleDevice> consumer) {
        while (true) { // NOSONAR
            try {
                final BlockingQueue<Event> events = subscribe(api);

                int noMessageCtr = 0;

                while (true) {
                    final Event event = events.poll(TIMEOUT, TimeUnit.MILLISECONDS);
                    if (event != null) {
                        noMessageCtr = 0;
                        if (event.getEvent().equals("devices")) {
                            final JSONObject devices = new JSONObject(event.getData());

                            devices.keySet().stream().map(id -> new MieleDevice(id, devices.getJSONObject(id)))
                                .forEach(consumer::accept);
                        } else if (event.getEvent().equals("ping")) {
                            LOGGER.debug(".");
                        }
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
            } catch (Exception e) {
                LOGGER.error(e.getMessage(), e);
                try {
                    // Wait one minute after error (e.g. Internet connection down)
                    Thread.sleep(2_000);
                    api.updateToken();
                } catch (InterruptedException interruptedException) {
                    interruptedException.printStackTrace();
                    Thread.currentThread().interrupt();
                    shutdown();
                    return;
                }
            }
        }
    }

}
