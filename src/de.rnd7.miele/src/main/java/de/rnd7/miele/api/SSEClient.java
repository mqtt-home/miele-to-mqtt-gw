package de.rnd7.miele.api;

import org.apache.client.sse.ApacheHttpSseClient;
import org.apache.client.sse.Event;
import org.apache.client.sse.SseRequest;
import org.apache.client.sse.SseResponse;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.impl.nio.client.HttpAsyncClients;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.*;

public class SSEClient {
    private static final Logger LOGGER = LoggerFactory.getLogger(SSEClient.class);
    private final ExecutorService executor = Executors.newFixedThreadPool(1);

    public BlockingQueue<Event> subscribe(MieleAPI api) throws ExecutionException, InterruptedException {
        final CloseableHttpAsyncClient asyncClient = HttpAsyncClients.createDefault();
        asyncClient.start();
        final SseRequest request = new SseRequest("https://api.mcs3.miele.com/v1/devices/all/events");
        request.setHeader("Accept-Language", "en-GB");
        request.setHeader("Authorization", "Bearer " + api.getToken().getAccessToken());
        final ApacheHttpSseClient sseClient = new ApacheHttpSseClient(asyncClient, executor);

        final SseResponse sseResponse = sseClient.execute(request).get();
        return sseResponse.getEntity().getEvents();
    }

    public void start(MieleAPI api) {
        while (true) { // NOSONAR
            try {
                api.updateToken();

                final BlockingQueue<Event> events = subscribe(api);

                while (true) {
                    Event event = events.poll(1, TimeUnit.MINUTES);
                    if (event == null) {
                        LOGGER.info("No event within 1m (Not even PING). Reconnecting.");
                        break;
                    }

                    if (event.getEvent().equals("devices")) {
                        LOGGER.info(event.getData());
                    }
                    else if (event.getEvent().equals("ping")) {
                        System.out.print(".");
                    }
                }
            } catch (Exception e) {
                LOGGER.error(e.getMessage(), e);
                try {
                    Thread.sleep(20000);
                } catch (InterruptedException interruptedException) {
                    interruptedException.printStackTrace();
                    Thread.currentThread().interrupt();
                }
            }
        }
    }

}
