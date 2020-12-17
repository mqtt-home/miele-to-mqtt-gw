package de.rnd7.mieletomqtt.miele;

import java.time.Duration;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;

import org.json.JSONObject;

import de.rnd7.mieletomqtt.mqtt.Message;

public class MieleDevice {
	private final String id;
	private final JSONObject data;
	private final int phaseId;
	private final ProgramPhase phase;
	private final State state;
	private final Duration remainingDuration;
	private final JSONObject small;

	public MieleDevice(final String id, final JSONObject data) {
		this.id = id;
		this.data = data;

		final JSONObject deviceState = data.getJSONObject("state");
		final JSONObject deviceStateProgramPhase = deviceState.getJSONObject("programPhase");

		this.phaseId = deviceStateProgramPhase.getInt("value_raw");
		this.phase = ProgramPhase.fromId(this.phaseId);

		final JSONObject deviceStateStatus = deviceState.getJSONObject("status");

		this.state = State.fromId(deviceStateStatus.getInt("value_raw"));
		this.remainingDuration = DurationParser.parse(deviceState.getJSONArray("remainingTime"));

		this.small = new JSONObject();

		if (this.state.equals(State.OFF)) {
			final String timeCompleted = LocalDateTime.now()
					.format(DateTimeFormatter.ofPattern("HH:mm"));

			this.small.put("remainingDurationMinutes", 0);
			this.small.put("remainingDuration", formatDuration(Duration.ZERO));
			this.small.put("timeCompleted", timeCompleted);
			this.small.put("phaseId", this.phaseId);
			this.small.put("phase", this.phase);
			this.small.put("state", this.state);
		}
		else {
			final String timeCompleted = LocalDateTime.now().plus(this.remainingDuration)
					.format(DateTimeFormatter.ofPattern("HH:mm"));

			this.small.put("remainingDurationMinutes", this.remainingDuration.toMinutes());
			this.small.put("remainingDuration", formatDuration(this.remainingDuration));
			this.small.put("timeCompleted", timeCompleted);
			this.small.put("phaseId", this.phaseId);
			this.small.put("phase", this.phase);
			this.small.put("state", this.state);
		}

	}

	private static String formatDuration(final Duration duration) {
		final long hours = duration.toHours();
		final long minutes = duration.minus(Duration.ofHours(hours)).toMinutes();
		return String.format("%d:%02d", hours, minutes);
	}

	public JSONObject getData() {
		return this.data;
	}

	public String getId() {
		return this.id;
	}

	public int getPhaseId() {
		return this.phaseId;
	}

	public ProgramPhase getPhase() {
		return this.phase;
	}

	public State getState() {
		return this.state;
	}

	public Duration getRemainingDuration() {
		return this.remainingDuration;
	}

	public Message toFullMessage() {
		return new Message(this.id + "/full", this.data);
	}

	public Message toSmallMessage() {
		return new Message(this.id, this.small);
	}
}
