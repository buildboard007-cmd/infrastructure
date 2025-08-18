import {StackOptions} from "../lib/types/stack-options";
import {StageEnvironment} from "../lib/types/stage-environment";

let stackName = "Infrastructure";

export const options: StackOptions = {
    defaultRegion: "us-east-2",
    toolsAccount: "401448503050",
    productionAccount: "186375394147",
    devAccount: "521805123898",
    cdkBootstrapQualifier: "mbb313cmk",
    pipelineName: `${stackName}-pipeline`,
    stackName: stackName,
    githubConnectionArn: "arn:aws:codeconnections:us-east-2:401448503050:connection/03339418-8c24-4619-816d-ee23651c4d12",
	githubBranch: "main",
	githubOwner: "buildboard007-cmd",
	githubRepo: "infrastructure",
    serviceName: "infrastructure",
    localStageOptions: {
        environment: StageEnvironment.LOCAL,
        account: "000000000000",
        logRetentionDays: 1,
    },
    stageOptions: [
        {
            environment: StageEnvironment.DEV,
            account: "521805123898",
            logRetentionDays: 1,
        },
        {
            environment: StageEnvironment.PROD,
            account: "186375394147",
            logRetentionDays: 1,
        },
    ],
};
