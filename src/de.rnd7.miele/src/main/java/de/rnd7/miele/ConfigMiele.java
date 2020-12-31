package de.rnd7.miele;

import com.google.gson.annotations.SerializedName;

public class ConfigMiele {
    public static enum Mode {
        sse,
        polling
    }

    @SerializedName("client-id")
    private String clientId;
    @SerializedName("client-secret")
    private String clientSecret;

    @SerializedName("username")
    private String username;
    @SerializedName("password")
    private String password;

    @SerializedName("mode")
    private Mode mode = Mode.polling;

    public String getClientId() {
        return clientId;
    }

    public ConfigMiele setClientId(final String clientId) {
        this.clientId = clientId;
        return this;
    }

    public String getClientSecret() {
        return clientSecret;
    }

    public ConfigMiele setClientSecret(final String clientSecret) {
        this.clientSecret = clientSecret;
        return this;
    }

    public String getPassword() {
        return password;
    }

    public ConfigMiele setPassword(final String password) {
        this.password = password;
        return this;
    }

    public String getUsername() {
        return username;
    }

    public ConfigMiele setUsername(final String username) {
        this.username = username;
        return this;
    }

    public Mode getMode() {
        return mode;
    }

    public ConfigMiele setMode(final Mode mode) {
        this.mode = mode;
        return this;
    }
}
