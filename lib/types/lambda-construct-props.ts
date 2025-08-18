import {StackOptions} from "./stack-options";
import {StageEnvironment} from "./stage-environment";
import {IKey} from "aws-cdk-lib/aws-kms";
import {OpenApiBuilder} from "openapi3-ts";
import {Table} from "aws-cdk-lib/aws-dynamodb";

export interface LambdaConstructProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
}
