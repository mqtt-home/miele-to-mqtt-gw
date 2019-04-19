package de.rnd7.mieletomqtt.config;

import java.time.Duration;

public class Config {
	private String mqttBroker;

	private Duration pollingInterval;
	private String fullMessageTopic;

	private String mieleClientId;
	private String mieleClientSecret;
	private String mieleUsername;
	private String mielePassword;

	private String timezone;

	public void setMqttBroker(final String mqttBroker) {
		this.mqttBroker = mqttBroker;
	}

	public String getMqttBroker() {
		return this.mqttBroker;
	}

	public void setPollingInterval(final Duration pollingInterval) {
		this.pollingInterval = pollingInterval;
	}

	public Duration getPollingInterval() {
		return this.pollingInterval;
	}

	public void setFullMessageTopic(final String fullMessageTopic) {
		this.fullMessageTopic = fullMessageTopic;
	}

	public String getFullMessageTopic() {
		return this.fullMessageTopic;
	}

	public void setMieleClientId(final String mieleClientId) {
		this.mieleClientId = mieleClientId;
	}

	public String getMieleClientId() {
		return this.mieleClientId;
	}

	public void setMieleClientSecret(final String mieleClientSecret) {
		this.mieleClientSecret = mieleClientSecret;
	}

	public String getMieleClientSecret() {
		return this.mieleClientSecret;
	}

	public void setMieleUsername(final String mieleUsername) {
		this.mieleUsername = mieleUsername;
	}

	public String getMieleUsername() {
		return this.mieleUsername;
	}

	public void setMielePassword(final String mielePassword) {
		this.mielePassword = mielePassword;
	}

	public String getMielePassword() {
		return this.mielePassword;
	}

	public void setTimezone(final String timezone) {
		this.timezone = timezone;
	}

	public String getTimezone() {
		return this.timezone;
	}

}
