package de.rnd7.mieletomqtt.miele;

import java.time.Duration;

import org.json.JSONArray;

public class DurationParser {
	public static Duration parse(final JSONArray array) {
		return Duration.ofHours(array.getInt(0)).plus(Duration.ofMinutes(array.getInt(1)));
	}
}
