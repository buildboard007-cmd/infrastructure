import * as cdk from "aws-cdk-lib";
import {StackProps} from "aws-cdk-lib";
import {Construct} from "constructs";
import {CodePipeline, CodePipelineSource, ManualApprovalStep, ShellStep,} from "aws-cdk-lib/pipelines";
import {PipelineAppStage} from "./pipeline-app-stage";
import {options} from "../../config/config";
import {BuildSpec, LinuxBuildImage} from "aws-cdk-lib/aws-codebuild";
import {StackOptions} from "../types/stack-options";
import {StageEnvironment} from "../types/stage-environment";

interface PipelineStackProps extends StackProps {
    options: StackOptions;
}

export class PipelineStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: PipelineStackProps) {
        super(scope, id, props);

        const pipeline = new CodePipeline(this, `CodePipeline`, {
            crossAccountKeys: true,
            selfMutation: true,
            pipelineName: `${props.options.pipelineName}`,
            dockerEnabledForSynth: true,
            synth: new ShellStep("Synth", {
                input: CodePipelineSource.connection(`${props.options.githubOwner}/${props.options.githubRepo}`, `main`, {connectionArn: props.options.githubConnectionArn ?? "",}),
                commands: [
                    "npm ci",
                    "npm run build",
                    "npx cdk synth",
                ],
            }),
            
            synthCodeBuildDefaults: {
                buildEnvironment: {buildImage: LinuxBuildImage.STANDARD_7_0},
                partialBuildSpec: BuildSpec.fromObject({
                    phases: {
                        install: {
                            "runtime-versions": {
                                golang: "1.24",
                            },
                        },
                    },
                }),
            },
        });

        for (const option of props.options.stageOptions ?? []) {
            const stage = new PipelineAppStage(this, option.environment, {
                options: options,
                env: {account: option.account, region: props.options.defaultRegion},
                stageEnvironment: option.environment,
            });
            const stageDeployment = pipeline.addStage(stage);
            if (option.environment !== StageEnvironment.DEV) {
                stageDeployment.addPre(
                    new ManualApprovalStep(`PromoteTo${option.environment}`)
                );
            }
            // stageDeployment.addPost(
            //     new InsertDataStep(this, `${option.environment}InsertDataStep`, {
            //         options: options,
            //         account: option.account,
            //     }).exportStep
            // );
        }
    }
}
