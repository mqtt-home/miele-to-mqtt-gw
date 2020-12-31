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
import java.util.concurrent.*;
import java.util.function.Consumer;

public class SSEClient {
    private static final Logger LOGGER = LoggerFactory.getLogger(SSEClient.class);
    private final ExecutorService executor = Executors.newFixedThreadPool(1);
    private CloseableHttpAsyncClient asyncClient;

    BlockingQueue<Event> subscribe(MieleAPI api) throws Exception {
        LOGGER.debug("Subscribe SSE");
        asyncClient = HttpAsyncClients.createDefault();
        asyncClient.start();
        final SseRequest request = new SseRequest("https://api.mcs3.miele.com/v1/devices/all/events");
        request.setHeader("Accept-Language", "en-GB");
        request.setHeader("Authorization", "Bearer " + api.getToken().getAccessToken());
        final ApacheHttpSseClient sseClient = new ApacheHttpSseClient(asyncClient, executor);

        final Future<SseResponse> future = sseClient.execute(request);

        final SseResponse sseResponse = future.get(10, TimeUnit.SECONDS);
        return sseResponse.getEntity().getEvents();
    }

    void testClose() throws IOException {
        asyncClient.close();
    }

    boolean isRunning() {
        return asyncClient.isRunning();
    }

    public void start(MieleAPI api, Consumer<MieleDevice> consumer) {
        while (true) { // NOSONAR
            try {
                final BlockingQueue<Event> events = subscribe(api);

                while (true) {
                    final Event event = events.poll(6, TimeUnit.SECONDS);
                    if (event == null) {
                        break;
                    }

                    if (event.getEvent().equals("devices")) {
                        final JSONObject devices = new JSONObject(event.getData());

                        devices.keySet().stream().map(id -> new MieleDevice(id, devices.getJSONObject(id)))
                                .forEach(consumer::accept);
                    }
                    else if (event.getEvent().equals("ping")) {
                        LOGGER.debug(".");
                    }
                }
            } catch (Exception e) {
                LOGGER.error(e.getMessage(), e);
                try {
                    // Wait one minute after error (e.g. Internet connection down)
                    Thread.sleep(60_000);
                    api.updateToken();
                } catch (InterruptedException interruptedException) {
                    interruptedException.printStackTrace();
                    Thread.currentThread().interrupt();
                }
            }
        }
    }

}
