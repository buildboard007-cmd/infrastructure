import * as cdk from "aws-cdk-lib";
import {Construct} from "constructs";
import {StackOptions} from "./types/stack-options";
import {StageEnvironment} from "./types/stage-environment";
import {SubStack} from "./resources/sub_stack/sub-stack";


interface LambdaStackProps extends cdk.StackProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
}

export class MainStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: LambdaStackProps) {
        super(scope, id, {
            ...props,
        });

        cdk.Tags.of(this).add("billingTag", props.options.serviceName);
        cdk.Tags.of(this).add("serviceName", props.options.serviceName);

        const subStack = new SubStack(this, `SubStack`, {
            options: props.options,
            stageEnvironment: props.stageEnvironment,
        });

        new cdk.CfnOutput(this, 'CorsLambdaArn', {
            value: subStack.corsLambdaArn,
            exportName: `api-gateway-cors-lambda-arn`,
            description: 'ARN of the CORS Lambda function for API Gateway'
        });
    }

}
