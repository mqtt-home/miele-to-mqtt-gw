package de.rnd7.mqtt;

import com.google.gson.annotations.SerializedName;

import java.time.Duration;
import java.util.Optional;

public class ConfigMqtt {
    private String url;
    private String username;
    private String password;
    private boolean retain = true;

    @SerializedName("message-interval")
    private Duration pollingInterval = Duration.ofMinutes(1);

    @SerializedName("full-message-topic")
    private String fullMessageTopic = "miele";

    @SerializedName("client-id")
    private String clientId;

    public static ConfigMqtt createFor(String host, int port, String fullMessageTopic) {
        final ConfigMqtt config = new ConfigMqtt();
        config.setBroker(host, port);
        config.fullMessageTopic = fullMessageTopic;
        return config;
    }

    public ConfigMqtt setBroker(String host, int port) {
        this.url = String.format("tcp://%s:%s", host, port);
        return this;
    }

    public String getUrl() {
        return url;
    }

    public Optional<String> getUsername() {
        return Optional.ofNullable(username);
    }

    public Optional<String> getPassword() {
        return Optional.ofNullable(password);
    }

    public String getFullMessageTopic() {
        return fullMessageTopic;
    }

    public Duration getPollingInterval() {
        return pollingInterval;
    }

    public ConfigMqtt setPollingInterval(final Duration pollingInterval) {
        this.pollingInterval = pollingInterval;
        return this;
    }

    public boolean isRetain() {
        return retain;
    }

    public Optional<String> getClientId() {
        return Optional.ofNullable(clientId);
    }

    public ConfigMqtt setClientId(final String clientId) {
        this.clientId = clientId;
        return this;
    }
}
