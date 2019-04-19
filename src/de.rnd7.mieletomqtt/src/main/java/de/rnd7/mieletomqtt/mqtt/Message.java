package de.rnd7.mieletomqtt.mqtt;

import org.json.JSONObject;

public class Message {
	private final String topic;
	private final JSONObject json;

	public Message(final String topic, final JSONObject json) {
		this.topic = topic;
		this.json = json;
	}

	public String getTopic() {
		return this.topic;
	}

	public JSONObject getJson() {
		return this.json;
	}
}
