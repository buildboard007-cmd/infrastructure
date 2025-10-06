import {StackOptions} from "./stack-options";
import {StageEnvironment} from "./stage-environment";
import * as s3 from "aws-cdk-lib/aws-s3";

export interface LambdaConstructProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
    attachmentBucket?: s3.Bucket;
}
