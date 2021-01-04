package de.rnd7.mieletomqtt.miele;

import de.rnd7.miele.api.MieleAPI;

import java.util.Objects;

class TestHelper {

    private TestHelper() {

    }

    static String forceEnv(String propName) {
        // macOS note: sudo vi /etc/launchd.conf
        final String value = Objects.requireNonNull(System.getenv(propName),
            String.format("ENV property %s is required to run this test case.", propName));

        if (value.trim().isEmpty()) {
            throw new IllegalArgumentException(String.format("ENV property %s must not be empty to run this test case.", propName));
        }

        return value;
    }

}
