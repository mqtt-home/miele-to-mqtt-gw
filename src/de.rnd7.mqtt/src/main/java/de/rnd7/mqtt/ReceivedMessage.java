package de.rnd7.mqtt;

import java.util.Objects;

public class ReceivedMessage {

	private final String topic;
	private final String data;

	public ReceivedMessage(final String topic, final String data) {
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
	public boolean equals(Object o) {
		if (this == o) return true;
		if (o == null || getClass() != o.getClass()) return false;
		ReceivedMessage message = (ReceivedMessage) o;
		return Objects.equals(topic, message.topic) &&
				Objects.equals(data, message.data);
	}

	@Override
	public int hashCode() {
		return Objects.hash(topic, data);
	}

	@Override
	public String toString() {
		return "ReceivedMessage{" +
				"topic='" + topic + '\'' +
				", data='" + data + '\'' +
				'}';
	}
}
