package de.rnd7.miele.api;

import java.util.Map;
import java.util.Optional;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public enum ProgramPhase {
    UNKNOWN(-1), OFF(0), NOT_RUNNING(1792), REACTIVATING(1793), PRE_WASH(1794), MAIN_WASH(1795), RINSE(1796),
    INTERIM_RINSE(1797), FINAL_RINSE(1798), DRYING(1799), FINISHED(1800), PRE_WASH2(1801);

    private static Map<Integer, ProgramPhase> lookup;

    static {
        lookup = Stream.of(ProgramPhase.values()).collect(Collectors.toMap(p -> p.id, p -> p));
    }

    private int id;

    private ProgramPhase(final int id) {
        this.id = id;
    }

    public static ProgramPhase fromId(final int id) {
        return Optional.ofNullable(lookup.get(id)).orElse(UNKNOWN);
    }

}
