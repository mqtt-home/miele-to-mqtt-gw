package org.apache.client.sse;

import org.apache.http.*;
import org.apache.http.message.BasicHttpResponse;

import java.util.Locale;

public class SseResponse extends BasicHttpResponse {

    private final HttpResponse original;
    private final SseEntity entity;

    public SseResponse(HttpResponse original) {
        super(original.getStatusLine());
        this.original = original;
        this.entity = new SseEntity(original.getEntity());
    }

    @Override
    public SseEntity getEntity() {
        return entity;
    }

    public String getLastEventId() {
        return entity.getLastEventId();
    }

    @Override
    public StatusLine getStatusLine() {
        return original.getStatusLine();
    }

    @Override
    public void setStatusLine(StatusLine statusline) {
        original.setStatusLine(statusline);
    }

    @Override
    public void setStatusLine(ProtocolVersion ver, int code) {
        original.setStatusLine(ver, code);
    }

    @Override
    public void setStatusLine(ProtocolVersion ver, int code, String reason) {
        original.setStatusLine(ver, code, reason);
    }

    @Override
    public void setStatusCode(int code) throws IllegalStateException {
        original.setStatusCode(code);
    }

    @Override
    public void setReasonPhrase(String reason) throws IllegalStateException {
        original.setReasonPhrase(reason);
    }

    @Override
    public Locale getLocale() {
        return original.getLocale();
    }

    @Override
    public void setLocale(Locale loc) {
        original.setLocale(loc);
    }

    @Override
    public ProtocolVersion getProtocolVersion() {
        return original.getProtocolVersion();
    }

    @Override
    public boolean containsHeader(String name) {
        return original.containsHeader(name);
    }

    @Override
    public Header[] getHeaders(String name) {
        return original.getHeaders(name);
    }

    @Override
    public Header getFirstHeader(String name) {
        return original.getFirstHeader(name);
    }

    @Override
    public Header getLastHeader(String name) {
        return original.getLastHeader(name);
    }

    @Override
    public Header[] getAllHeaders() {
        return original.getAllHeaders();
    }

    @Override
    public void addHeader(Header header) {
        original.addHeader(header);
    }

    @Override
    public void addHeader(String name, String value) {
        original.addHeader(name, value);
    }

    @Override
    public void setHeader(Header header) {
        original.setHeader(header);
    }

    @Override
    public void setHeader(String name, String value) {
        original.setHeader(name, value);
    }

    @Override
    public void setHeaders(Header[] headers) {
        original.setHeaders(headers);
    }

    @Override
    public void removeHeader(Header header) {
        original.removeHeader(header);
    }

    @Override
    public void removeHeaders(String name) {
        original.removeHeaders(name);
    }

    @Override
    public HeaderIterator headerIterator() {
        return original.headerIterator();
    }

    @Override
    public HeaderIterator headerIterator(String name) {
        return original.headerIterator(name);
    }
}
