package org.apache.client.sse;

import org.apache.http.client.methods.HttpGet;

import java.net.URI;

/**
 * Allows us to set the correct Accept header automatically and always use HTTP GET.
 */
public class SseRequest extends HttpGet {

    public SseRequest() {
        addHeader("Accept", "text/event-stream");
    }

    public SseRequest(final URI uri) {
        super(uri);
        addHeader("Accept", "text/event-stream");
    }

    public SseRequest(final String uri) {
        super(uri);
        addHeader("Accept", "text/event-stream");
    }
}
