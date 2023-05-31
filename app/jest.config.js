module.exports = {
    coverageThreshold: {
        global: {
            branches: 80,
            functions: 85,
            lines: 85,
            statements: 85
        }
    },
    modulePathIgnorePatterns: [
        "<rootDir>/dist/"
    ],
    coverageDirectory: "build_internal/test_results",
    reporters: ["jest-standard-reporter", "jest-junit"],
    collectCoverageFrom: [
        "src/**/*.{ts,tsx,js,jsx}",
        "lib/**/*.{ts,tsx,js,jsx}"
    ],
    transform: {
        "^.+\\.(ts|tsx|js|jsx)$": "ts-jest",
    },
    setupFiles: ["<rootDir>/test/setup.ts"],
}
