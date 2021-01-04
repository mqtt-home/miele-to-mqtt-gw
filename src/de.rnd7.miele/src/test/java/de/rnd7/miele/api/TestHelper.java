package de.rnd7.miele.api;

import de.rnd7.miele.ConfigMiele;

import java.util.Objects;

public class TestHelper {

    private TestHelper() {

    }

    private static String forceEnv(String propName) {
        // macOS note: sudo vi /etc/launchd.conf
        final String value = Objects.requireNonNull(System.getenv(propName),
            String.format("ENV property %s is required to run this test case.", propName));

        if (value.trim().isEmpty()) {
            throw new IllegalArgumentException(String.format("ENV property %s must not be empty to run this test case.", propName));
        }

        return value;
    }

    public static MieleAPI createAPI() {
        ConfigMiele configMiele = createConfig();

        return new MieleAPI(configMiele);
    }

    public static ConfigMiele createConfig() {
        return new ConfigMiele()
            .setClientId(forceEnv("MIELE_CLIENT_ID"))
            .setClientSecret(forceEnv("MIELE_CLIENT_SECRET"))
            .setUsername(forceEnv("MIELE_USERNAME"))
            .setPassword(forceEnv("MIELE_PASSWORD"));
    }
}
