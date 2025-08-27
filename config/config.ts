import {StackOptions} from "../lib/types/stack-options";
import {StageEnvironment} from "../lib/types/stage-environment";

let stackName = "Infrastructure";

export const options: StackOptions = {
    defaultRegion: "us-east-2",
    toolsAccount: "401448503050",
    productionAccount: "186375394147",
    devAccount: "521805123898",
    cdkBootstrapQualifier: "hnb659fds",
    pipelineName: `${stackName}-pipeline`,
    stackName: stackName,
    apiName: "Infrastructure API",
    apiStageName: "main",
    githubConnectionArn: "arn:aws:codeconnections:us-east-2:401448503050:connection/03339418-8c24-4619-816d-ee23651c4d12",
    githubBranch: "main",
    githubOwner: "buildboard007-cmd",
    githubRepo: "infrastructure",
    serviceName: "infrastructure",
    callbackUrls: {
        [StageEnvironment.LOCAL]: [
            "http://localhost:3000/callback",
        ],
        [StageEnvironment.DEV]: [
            "http://localhost:3000/callback",
            "https://dev.buildboard.com/callback",
            "https://oauth.pstmn.io/v1/callback", // Postman for testing
        ],
        [StageEnvironment.PROD]: [
            "https://buildboard.com/callback",
        ]
    },
    logoutUrls: {
        [StageEnvironment.LOCAL]: [
            "http://localhost:3000/callback",
        ],
        [StageEnvironment.DEV]: [
            "http://localhost:3000",
            "https://dev.buildboard.com/callback",
            "https://dev.buildboard.com/callback",
        ],
        [StageEnvironment.PROD]: [
            "https://buildboard.com",
        ],
    },
    isTemporaryStack: false,
    localStageOptions: {
        environment: StageEnvironment.LOCAL,
        account: "000000000000",
        vpcId: "",
        domainName: "",
        domainNameAliasHostedZoneId: "",
        domainNameAliasTarget: "",
        logRetentionDays: 1,
    },
    stageOptions: [
        {
            environment: StageEnvironment.DEV,
            account: "521805123898",
            logRetentionDays: 1,
            vpcId: "",
            domainName: "",
            domainNameAliasHostedZoneId: "",
            domainNameAliasTarget: "",
        },
        {
            environment: StageEnvironment.PROD,
            account: "186375394147",
            logRetentionDays: 1,
            vpcId: "",
            domainName: "",
            domainNameAliasHostedZoneId: "",
            domainNameAliasTarget: "",
        },
    ],
};
