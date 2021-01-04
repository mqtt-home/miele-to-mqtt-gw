package org.apache.client.sse;

import org.apache.http.HttpResponse;
import org.apache.http.client.methods.HttpUriRequest;
import org.apache.http.concurrent.FutureCallback;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.nio.IOControl;
import org.apache.http.nio.client.methods.AsyncCharConsumer;
import org.apache.http.nio.client.methods.HttpAsyncMethods;
import org.apache.http.protocol.HttpContext;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.nio.CharBuffer;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutorService;

/**
 * https://github.com/manpreet333/apache-sseclient
 * <p>
 * Wraps the Async client and executes the get request to start listening to event stream in a new thread.
 * To abort the connection from client side, async client should be closed.
 */
public class ApacheHttpSseClient {
    private static final Logger LOGGER = LoggerFactory.getLogger(ApacheHttpSseClient.class);

    private final CloseableHttpAsyncClient httpAsyncClient;
    private final ExecutorService executorService;

    public ApacheHttpSseClient(final CloseableHttpAsyncClient httpAsyncClient, final ExecutorService executorService) {
        this.httpAsyncClient = httpAsyncClient;
        this.executorService = executorService;
    }

    public CompletableFuture<SseResponse> execute(final HttpUriRequest request) {
        final CompletableFuture<SseResponse> futureResp = new CompletableFuture<>();
        final AsyncCharConsumer<SseResponse> charConsumer = new AsyncCharConsumer<SseResponse>() { // NOSONAR
            private SseResponse response;

            @Override
            protected void onCharReceived(final CharBuffer buf, final IOControl ioctrl) throws IOException {
                //Push chars buffer to entity for parsing and storage
                response.getEntity().pushBuffer(buf, ioctrl);
            }

            @Override
            protected void onResponseReceived(final HttpResponse response) {
                this.response = new SseResponse(response);
                futureResp.complete(this.response);
            }

            @Override
            protected SseResponse buildResult(final HttpContext context) throws Exception {
                return response;
            }
        };

        final FutureCallback<SseResponse> callback = new FutureCallback<SseResponse>() {
            @Override
            public void completed(final SseResponse result) {
                futureResp.cancel(true);
                closeQuietly(charConsumer);
            }

            @Override
            public void failed(final Exception excObj) {
                LOGGER.error(excObj.getMessage(), excObj);
                futureResp.completeExceptionally(excObj);
                closeQuietly(charConsumer);
            }

            @Override
            public void cancelled() {
                futureResp.cancel(true);
                closeQuietly(charConsumer);
            }
        };

        executorService.submit(() ->
            httpAsyncClient.execute(HttpAsyncMethods.create(request), charConsumer, callback)
        );
        return futureResp;
    }

    private void closeQuietly(final AsyncCharConsumer<SseResponse> charConsumer) {
        try {
            charConsumer.close();
        } catch (IOException e) {
            LOGGER.trace(e.getMessage(), e);
        }
    }
}
