package de.rnd7.miele;

import com.google.gson.annotations.SerializedName;

public class ConfigMiele {
    @SerializedName("client-id")
    private String clientId;
    @SerializedName("client-secret")
    private String clientSecret;

    @SerializedName("username")
    private String username;
    @SerializedName("password")
    private String password;

    public String getClientId() {
        return clientId;
    }

    public String getClientSecret() {
        return clientSecret;
    }

    public String getPassword() {
        return password;
    }

    public String getUsername() {
        return username;
    }

}
