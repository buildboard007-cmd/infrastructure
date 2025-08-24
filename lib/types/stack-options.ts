import {StageEnvironment} from "./stage-environment";

export type StackOptions = {
    defaultRegion: string,
    toolsAccount: string,
    devAccount: string,
    productionAccount: string,
    cdkBootstrapQualifier: string,
    stackName: string,
    pipelineName: string,
    githubConnectionArn: string,
    githubBranch: string,
    githubOwner: string,
    githubRepo: string,
    serviceName: string,
    callbackUrls: {
        [key in StageEnvironment]: string[];
    },
    logoutUrls: {
        [key in StageEnvironment]: string[];
    },
    isTemporaryStack: boolean,
    localStageOptions: StageOptions,
    stageOptions: StageOptions[],
};

export type StageOptions = {
    environment: StageEnvironment,
    account: string,
    logRetentionDays: number
};

export const findStageOption = (options: StackOptions, o: StageEnvironment) => {
    if (o === StageEnvironment.LOCAL) {
        return options.localStageOptions;
    }

    const stageOption = options.stageOptions.find((option) => option.environment == o);
    if (!stageOption) {
        throw new Error(`Could not find account for ${o}`);
    }

    return stageOption;
};
