package org.apache.client.sse;

import org.apache.http.HttpResponse;
import org.apache.http.client.methods.HttpUriRequest;
import org.apache.http.concurrent.FutureCallback;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.nio.IOControl;
import org.apache.http.nio.client.methods.AsyncCharConsumer;
import org.apache.http.nio.client.methods.HttpAsyncMethods;
import org.apache.http.protocol.HttpContext;

import java.io.IOException;
import java.nio.CharBuffer;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Future;

/**
 * https://github.com/manpreet333/apache-sseclient
 *
 * Wraps the Async client and executes the get request to start listening to event stream in a new thread.
 * To abort the connection from client side, async client should be closed.
 */
public class ApacheHttpSseClient {

    private final CloseableHttpAsyncClient httpAsyncClient;
    private final ExecutorService executorService;

    public ApacheHttpSseClient(CloseableHttpAsyncClient httpAsyncClient, ExecutorService executorService) {
        this.httpAsyncClient = httpAsyncClient;
        this.executorService = executorService;
    }

    public Future<SseResponse> execute(HttpUriRequest request) {

        CompletableFuture<SseResponse> futureResp = new CompletableFuture<>();
        AsyncCharConsumer<SseResponse> charConsumer = new AsyncCharConsumer<SseResponse>() {
            private SseResponse response;
            @Override
            protected void onCharReceived(CharBuffer buf, IOControl ioctrl) throws IOException {
                //Push chars buffer to entity for parsing and storage
                response.getEntity().pushBuffer(buf, ioctrl);
            }

            @Override
            protected void onResponseReceived(HttpResponse response) {
                this.response = new SseResponse(response);
                futureResp.complete(this.response);
            }

            @Override
            protected SseResponse buildResult(HttpContext context) throws Exception {
                return response;
            }
        };

        executorService.submit(() ->
                httpAsyncClient.execute(HttpAsyncMethods.create(request), charConsumer, new FutureCallback<SseResponse>() {
                        @Override
                        public void completed(SseResponse result) {
                            System.out.println("completed");
                            futureResp.complete(result);
                        }

                        @Override
                        public void failed(Exception excObj) {
                            System.out.println(excObj.getMessage());
                            futureResp.completeExceptionally(excObj);
                        }

                        @Override
                        public void cancelled() {
                            System.out.println("cancelled");
                            futureResp.cancel(true);
                        }
                    }
                )
        );
        return futureResp;
    }

}