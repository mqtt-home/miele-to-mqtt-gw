package de.rnd7.miele.api;

import org.apache.client.sse.Event;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.TimeUnit;

import static org.awaitility.Awaitility.await;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

public class SSEClientTest {
    @Test
    public void test_sse() throws Exception {
        final SSEClient client = new SSEClient();

        final BlockingQueue<Event> events = client.subscribe(TestHelper.createAPI());

        // Expect initial event to raise immediately
        final Event first = events.poll(1, TimeUnit.SECONDS);
        assertNotNull(first);
        assertEquals("devices", first.getEvent());
        assertNotNull(first.getData());

        // Expect second event to raise after around 5 seconds (PING or device data)
        final Event second = events.poll(10, TimeUnit.SECONDS);
        assertNotNull(second);
    }

    @Test
    public void test_sse_reconnect() throws Exception {
        final SSEClient client = new SSEClient();
        final List<MieleDevice> devices = new ArrayList<>();

        new Thread(() -> client.start(TestHelper.createAPI(),
            device -> {
                synchronized (devices) {
                    devices.add(device);
                }
            })).start();

        for (int i = 0; i < 3; i++) {
            await().atMost(Duration.ofSeconds(5)).until(() -> !devices.isEmpty());

            // Expect initial event to raise immediately
            final MieleDevice first = devices.get(0);
            assertNotNull(first);
            synchronized (devices) {
                client.closeClient();
                await().atMost(Duration.ofSeconds(10)).until(client::isRunning);
                devices.clear();
            }
        }

    }
}
