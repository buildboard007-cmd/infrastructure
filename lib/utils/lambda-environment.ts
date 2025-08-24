/**
 * Centralized environment configuration for Lambda functions
 * Provides consistent environment variables across all Lambda functions
 */

import { StageEnvironment } from "../types/stage-environment";

/**
 * Get base environment variables for all Lambda functions
 * These are the common environment variables that every Lambda should have
 */
export function getBaseLambdaEnvironment(stageEnvironment: StageEnvironment): { [key: string]: string } {
    return {
        LOG_LEVEL: "error",  // Cost optimization - only log errors by default
        IS_LOCAL: "false",
        ENVIRONMENT: stageEnvironment,
    };
}

/**
 * Get environment variables for database Lambda functions
 * Includes base environment plus database-specific settings
 */
export function getDatabaseLambdaEnvironment(stageEnvironment: StageEnvironment): { [key: string]: string } {
    return {
        ...getBaseLambdaEnvironment(stageEnvironment),
        // Add any database-specific environment variables here if needed
    };
}

/**
 * Merge custom environment variables with base environment
 * @param baseEnv Base environment variables
 * @param customEnv Custom environment variables to add/override
 */
export function mergeEnvironment(
    baseEnv: { [key: string]: string },
    customEnv?: { [key: string]: string }
): { [key: string]: string } {
    return {
        ...baseEnv,
        ...(customEnv || {})
    };
}