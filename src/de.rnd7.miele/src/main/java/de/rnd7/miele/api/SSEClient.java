package de.rnd7.miele.api;

import org.apache.client.sse.ApacheHttpSseClient;
import org.apache.client.sse.Event;
import org.apache.client.sse.SseRequest;
import org.apache.client.sse.SseResponse;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.impl.nio.client.HttpAsyncClients;
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
        closeClient();

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

    public void start(final MieleAPI api, final Consumer<MieleDevice> consumer) {
        while (true) { // NOSONAR
            try {
                new SSEClientHandler(subscribe(api), asyncClient, consumer).run();
            } catch (Exception e) {
                LOGGER.error(e.getMessage(), e);
                try {
                    // Wait one minute after error (e.g. Internet connection down)
                    Thread.sleep(60_000);
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
