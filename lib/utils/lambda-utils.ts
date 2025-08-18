import {FuncProps} from "../types/func-props";

export function GetRetentionDays(props: FuncProps) {

    // Get parameter for log retention
    let logRetention = props?.options.stageOptions.find(stg => stg.environment === props?.stageEnvironment)?.logRetentionDays

    // Set default if not exists
    return logRetention || 1

}