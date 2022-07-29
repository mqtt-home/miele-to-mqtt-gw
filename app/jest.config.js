module.exports = {
    coverageThreshold: {
        global: {
            branches: 70,
            functions: 80,
            lines: 80,
            statements: 80
        }
    },
    modulePathIgnorePatterns: [
        "<rootDir>/dist/"
    ],
    coverageDirectory: "build_internal/test_results",
    reporters: ["jest-standard-reporter", "jest-junit"],
    collectCoverage: true,
    collectCoverageFrom:  [
        "src/**/*.{ts,tsx,js,jsx}",
        "lib/**/*.{ts,tsx,js,jsx}"
    ],
    transform: {
        "^.+\\.(ts|tsx|js|jsx)$": "ts-jest",
    },
    setupFiles: ["<rootDir>/test/setup.ts"],
}
