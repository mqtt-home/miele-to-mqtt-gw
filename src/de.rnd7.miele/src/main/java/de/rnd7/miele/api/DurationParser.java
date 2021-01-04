package de.rnd7.miele.api;

import java.time.Duration;

import org.json.JSONArray;

public class DurationParser {

    private DurationParser() {

    }

    public static Duration parse(final JSONArray array) {
        if (array == null) {
            return Duration.ZERO;
        } else if (array.length() == 2) {
            return Duration.ofHours(array.getInt(0))
                .plus(Duration.ofMinutes(array.getInt(1)));
        } else if (array.length() == 3) {
            return Duration.ofHours(array.getInt(0))
                .plus(Duration.ofMinutes(array.getInt(1)))
                .plus(Duration.ofSeconds(array.getInt(2)));
        } else {
            return Duration.ZERO;
        }
    }
}
