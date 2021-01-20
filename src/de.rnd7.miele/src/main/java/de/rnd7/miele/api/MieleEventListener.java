package de.rnd7.miele.api;

public interface MieleEventListener {
    String MIELE_STATE = "bridge/miele";

    void accept(final MieleDevice mieleDevice);

    default void state(final MieleAPIState state) {};
}
