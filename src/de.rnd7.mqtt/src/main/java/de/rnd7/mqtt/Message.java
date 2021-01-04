package de.rnd7.mqtt;

import java.util.Objects;

public class Message {
    private final String topic;
    private final String data;

    public Message(final String topic, final String data) {
        this.topic = topic;
        this.data = data;
    }

    public String getTopic() {
        return this.topic;
    }

    public String getData() {
        return this.data;
    }

    @Override
    public boolean equals(final Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        final Message message = (Message) o;
        return Objects.equals(topic, message.topic) &&
            Objects.equals(data, message.data);
    }

    @Override
    public int hashCode() {
        return Objects.hash(topic, data);
    }
}
