package org.apache.client.sse;

public class Event {

    private final String id;
    private final String event;
    private final String data;
    private final int retry;

    public Event(String id, String event, String data, int retry) {
        this.id = id;
        this.event = event;
        this.data = data;
        this.retry = retry;
    }

    public String getId() {
        return id;
    }

    public String getEvent() {
        return event;
    }

    public String getData() {
        return data;
    }

    public int getRetry() {
        return retry;
    }

    @Override
    public String toString() {
        StringBuilder eventString = new StringBuilder();
        if (id != null && id.length() > 0) {
            eventString.append("id: ");
            eventString.append(id);
        }
        if (event != null && event.length() > 0) {
            eventString.append("\nevent: ");
            eventString.append(event);
        }
        if (data != null && data.length() > 0) {
            eventString.append("\ndata: ");
            eventString.append(data);
        }
        if (retry != 0) {
            eventString.append("\nretry: ");
            eventString.append(retry);
        }
        return eventString.toString();
    }
}
