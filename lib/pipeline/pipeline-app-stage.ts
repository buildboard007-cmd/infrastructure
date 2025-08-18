import * as cdk from "aws-cdk-lib";
import {Construct} from "constructs";
import {MainStack} from "../main-stack";
import {StageEnvironment} from "../types/stage-environment";
import {StackOptions} from "../types/stack-options";

interface PipelineAppStageProps extends cdk.StageProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
}

export class PipelineAppStage extends cdk.Stage {
    constructor(scope: Construct, id: string, props: PipelineAppStageProps) {
        super(scope, id, props);

        new MainStack(
            this,
            `${props?.options.stackName}-AppStage`,
            {
                options: props.options,
                stageEnvironment: props.stageEnvironment,
            }
        );
    }
}
