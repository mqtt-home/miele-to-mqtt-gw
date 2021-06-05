package de.rnd7.miele.api;

import dev.manpreet.apache.sseclient.ApacheHttpSseClient;
import dev.manpreet.apache.sseclient.Event;
import dev.manpreet.apache.sseclient.SseRequest;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.impl.nio.client.HttpAsyncClients;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

public class SSEClient {
    private static final Logger LOGGER = LoggerFactory.getLogger(SSEClient.class);
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

        return new ApacheHttpSseClient(asyncClient, executor)
            .execute(request)
            .get(10, TimeUnit.SECONDS)
            .getEntity()
            .getEvents();
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

    public void start(final MieleAPI api, final MieleEventListener listener) {
        while (true) { // NOSONAR
            try {
                final BlockingQueue<Event> events = subscribe(api);
                listener.state(MieleAPIState.connected);

                new SSEClientHandler(events, asyncClient, listener::accept).run();
            } catch (Exception e) {
                listener.state(MieleAPIState.disconnected);
                LOGGER.error(e.getMessage(), e);

                if (!api.waitReconnect()) {
                    shutdown();
                    return;
                }
            }
        }
    }
}
