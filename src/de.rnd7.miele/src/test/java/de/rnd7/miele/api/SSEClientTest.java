package de.rnd7.miele.api;

import junit.framework.TestCase;
import org.apache.client.sse.Event;
import org.apache.client.sse.SseResponse;
import org.junit.Test;

import java.util.concurrent.BlockingQueue;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

public class SSEClientTest extends TestCase {
    @Test
    public void test_sse() throws Exception {
        SSEClient client = new SSEClient();

        final BlockingQueue<Event> events = client.subscribe(TestHelper.createAPI());

        // Expect initial event to raise immediately
        final Event first = events.poll(1, TimeUnit.SECONDS);
        assertNotNull(first);
        assertEquals("devices", first.getEvent());
        assertNotNull(first.getData());

        // Expect second event to raise after around 5 seconds (PING or device data)
        final Event second = events.poll(6, TimeUnit.SECONDS);
        assertNotNull(second);
    }
}