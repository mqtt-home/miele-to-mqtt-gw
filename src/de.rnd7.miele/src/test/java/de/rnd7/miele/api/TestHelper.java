package de.rnd7.miele.api;

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
        return new MieleAPI(
                forceEnv("MIELE_CLIENT_ID"),
                forceEnv("MIELE_CLIENT_SECRET"),
                forceEnv("MIELE_USERNAME"),
                forceEnv("MIELE_PASSWORD")
        );
    }
}
