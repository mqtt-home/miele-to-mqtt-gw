package de.rnd7.mieletomqtt.miele;

import java.util.Map;
import java.util.Optional;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public enum State {
	UNKNOWN(-1),
	RESERVED(0),
	OFF(1),
	ON(2),
	PROGRAMMED(3),
	PROGRAMMED_WAITING_TO_START (4),
	RUNNING(5),
	PAUSE(6),
	END_PROGRAMMED(7),
	FAILURE(8),
	PROGRAMME_INTERRUPTED(9),
	IDLE(10),
	RINSE_HOLD(11),
	SERVICE(12),
	SUPERFREEZING(13),
	SUPERCOOLING(14),
	SUPERHEATING(15);
	
	private static Map<Integer, State> lookup;
	
	static {
		lookup = Stream.of(State.values()).collect(Collectors.toMap(p -> p.id, p -> p));
	}
	
	private int id;

	private State(int id) {
		this.id = id;
	}

	public static State fromId(int id) {
		return Optional.ofNullable(lookup.get(id)).orElse(UNKNOWN);
	}
}
